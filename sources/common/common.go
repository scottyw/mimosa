package common

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"strings"
	"time"

	"cloud.google.com/go/pubsub"

	"cloud.google.com/go/storage"
	"google.golang.org/api/iterator"
)

// PubsubHandlerFunc captures the signature for a Cloud Function that can handle pubsub messages
type PubsubHandlerFunc func(context.Context, *pubsub.Message) error

type queryFunc func(config map[string]string) (map[string]MimosaData, error)

// MimosaData for an item (e.g. a host) it's binary data along with its Version and Typ (e.g. aws-instance v1.2)
type MimosaData struct {
	Version string
	Typ     string
	Data    []byte
}

// Build a Cloud Function from the  supplied query function
func Build(queryFunc queryFunc) PubsubHandlerFunc {
	// This is the pubsub handler
	return func(ctx context.Context, m *pubsub.Message) error {
		defer LogTiming(time.Now(), "Collect")

		bucketName := string(m.Data)
		if !strings.HasPrefix(bucketName, "source-") {
			return fmt.Errorf("message must be a bucket name starting with 'source-': %v", bucketName)
		}

		// Create GCP client and get a handle on the bucket
		client, err := storage.NewClient(context.Background())
		if err != nil {
			return err
		}
		bucket := client.Bucket(bucketName)

		// Load source config from the bucket
		var config map[string]string
		err = unmarshalFromBucket(bucket, "config.json", &config)
		if err != nil {
			return fmt.Errorf("Cannot read config.json: %v", err)
		}

		// Load state from previous runs
		var checksums map[string]string
		err = unmarshalFromBucket(bucket, "state.json", &checksums)
		if err != nil {
			if err != storage.ErrObjectNotExist {
				return fmt.Errorf("Cannot read state.json: %v", err)
			}
			// Use a default empty value instead
			checksums = map[string]string{}
		}

		// Collect API data
		items, err := queryFunc(config)
		if err != nil {
			return err
		}

		// Write items to the bucket
		for id, item := range items {
			// Only write this instance if it has changed
			start := time.Now()
			previousChecksum, present := checksums[id]
			sha := sha1.New()
			_, err = sha.Write(item.Data)
			if err != nil {
				log.Printf("failed to compute SHA: %v", err)
				continue
			}
			checksum := hex.EncodeToString(sha.Sum(nil))
			if !present || checksum != previousChecksum {
				err = writeToBucket(bucket, id, item.Typ, item.Version, item.Data)
				if err != nil {
					return err
				}
				checksums[id] = checksum
				log.Printf("Change: %s", id)
				log.Printf("Timing: Write: %dms", uint(time.Since(start).Seconds()*1000)) // Milliseconds not supported in Go 1.11
			} else {
				log.Printf("No change found: %s", id)
			}
		}
		if err := pruneBucket(bucket, items); err != nil {
			return err
		}
		checksums = pruneChecksums(checksums, items)
		// Write state back to the bucket
		data, err := json.Marshal(checksums)
		if err != nil {
			return fmt.Errorf("Cannot marshal the value: %v", err)
		}
		err = writeToBucket(bucket, "state.json", "", "", data)
		if err != nil {
			return err
		}

		return nil
	}
}

// Build_Iot a Cloud Function from the  supplied query function
func Build_Iot() PubsubHandlerFunc {
	// This is the pubsub handler
	return func(ctx context.Context, m *pubsub.Message) error {
		defer LogTiming(time.Now(), "Collect")

		// data:{"name":"10.16.118.74","privateIPv4":"10.16.118.74","privateIPv6":""}

		bucketName := m.Attributes["deviceId"]
		if !strings.HasPrefix(bucketName, "source-") {
			return fmt.Errorf("message must be a bucket name starting with 'source-': %v", bucketName)
		}

		// Create GCP client and get a handle on the bucket
		client, err := storage.NewClient(context.Background())
		if err != nil {
			return err
		}
		bucket := client.Bucket(bucketName)

		// Load source config from the bucket
		var config map[string]string
		err = unmarshalFromBucket(bucket, "config.json", &config)
		if err != nil {
			return fmt.Errorf("Cannot read config.json: %v", err)
		}

		// Load state from previous runs
		var checksums map[string]string
		err = unmarshalFromBucket(bucket, "state.json", &checksums)
		if err != nil {
			if err != storage.ErrObjectNotExist {
				return fmt.Errorf("Cannot read state.json: %v", err)
			}
			// Use a default empty value instead
			checksums = map[string]string{}
		}

		var hostData = make(map[string]string)
		err = json.Unmarshal(m.Data, &hostData)
		if err != nil {
			return err
		}

		// Write items to the bucket
		var item = MimosaData{
			Version: "1.0",
			Typ:     "netscan-instance",
			Data:    m.Data,
		}
		start := time.Now()
		id := hostData["name"]
		previousChecksum, present := checksums[id]
		sha := sha1.New()
		_, err = sha.Write(m.Data)
		if err != nil {
			log.Printf("failed to compute SHA: %v", err)
			return err
		}
		checksum := hex.EncodeToString(sha.Sum(nil))
		if !present || checksum != previousChecksum {
			err = writeToBucket(bucket, id, item.Typ, item.Version, item.Data)
			if err != nil {
				return err
			}
			checksums[id] = checksum
			log.Printf("Change: %s", id)
			log.Printf("Timing: Write: %dms", uint(time.Since(start).Seconds()*1000)) // Milliseconds not supported in Go 1.11
		} else {
			log.Printf("No change found: %s", id)
		}

		// Write state back to the bucket
		data, err := json.Marshal(checksums)
		if err != nil {
			return fmt.Errorf("Cannot marshal the value: %v", err)
		}
		err = writeToBucket(bucket, "state.json", "", "", data)
		if err != nil {
			return err
		}

		return nil
	}
}

// LogTiming logs an elapsed time
func LogTiming(start time.Time, name string) {
	log.Printf("Timing: %s: %dms", name, uint(time.Since(start).Seconds()*1000)) // Milliseconds not supported in Go 1.11
}

func writeToBucket(bucket *storage.BucketHandle, object string, typ string, version string, data []byte) error {
	defer LogTiming(time.Now(), "writeToBucket")
	oh := bucket.Object(object)
	wc := oh.NewWriter(context.Background())
	if typ != "" && version != "" {
		wc.ObjectAttrs.Metadata = map[string]string{
			"mimosa-type":         typ,
			"mimosa-type-version": version,
		}
	} else {
		if typ != "" || version != "" {
			return fmt.Errorf("Both, or neither, type and version must be specified (type is %v and version is %v)", typ, version)
		}
	}
	_, err := wc.Write(data)
	if err != nil {
		return err
	}
	err = wc.Close()
	if err != nil {
		return err
	}
	return nil
}

func unmarshalFromBucket(bucket *storage.BucketHandle, object string, v interface{}) error {
	defer LogTiming(time.Now(), "unmarshalFromBucket")
	rc, err := bucket.Object(object).NewReader(context.Background())
	if err != nil {
		return err
	}
	defer rc.Close()
	data, err := ioutil.ReadAll(rc)
	if err != nil {
		return err
	}
	err = json.Unmarshal(data, v)
	if err != nil {
		return err
	}
	return nil
}

func deleteFromBucket(bucket *storage.BucketHandle, object string) error {
	defer LogTiming(time.Now(), "deleteObject")
	log.Printf("Deleting: %s", object)
	oh := bucket.Object(object)
	return oh.Delete(context.Background())
}

func pruneChecksums(checksums map[string]string, items map[string]MimosaData) map[string]string {
	for k := range checksums {
		if _, present := items[k]; !present {
			log.Printf("Deleting checksum: %s", k)
			delete(checksums, k)
		}
	}
	return checksums
}
func pruneBucket(bucket *storage.BucketHandle, items map[string]MimosaData) error {
	//list everything in the bucket and check it's not in items, then delete if so
	it := bucket.Objects(context.Background(), nil)
	for {
		attrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return err
		}
		id := attrs.Name

		//FIXME I'm unsure why we have empty metadata on the state.json
		typ, hasType := attrs.Metadata["mimosa-type"]
		hasType = hasType && len(typ) > 0
		version, hasVersion := attrs.Metadata["mimosa-type-version"]
		hasVersion = hasVersion && len(version) > 0
		if !hasType || !hasVersion {
			// skip this one if it has insufficient metadata e.g. it's probably state.json or config.json
			continue
		}

		if _, present := items[id]; !present {
			err := deleteFromBucket(bucket, id)
			if err != nil {
				//consciously swallow this error to continue processing
				log.Printf("Error deleting object %v: %v ", attrs.Name, err)
			}
		}
	}
	return nil
}
