package gcpfinalize

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

	"cloud.google.com/go/firestore"
	"cloud.google.com/go/functions/metadata"
	"cloud.google.com/go/storage"
	compute "google.golang.org/api/compute/v1"
)

// GCSEvent is the payload of a GCS event.
type GCSEvent struct {
	Bucket         string    `json:"bucket"`
	Name           string    `json:"name"`
	Metageneration string    `json:"metageneration"`
	ResourceState  string    `json:"resourceState"`
	TimeCreated    time.Time `json:"timeCreated"`
	Updated        time.Time `json:"updated"`
}

// HandleInstance handles an GCSEvent and looks for a GCP compute instance
// gcloud functions deploy HandleInstance --runtime go111 --trigger-resource markf-test-bucket --trigger-event google.storage.object.finalize
func HandleInstance(ctx context.Context, e GCSEvent) error {
	meta, err := metadata.FromContext(ctx)
	if err != nil {
		return fmt.Errorf("metadata.FromContext: %v", err)
	}
	log.Printf("Event ID: %v\n", meta.EventID)
	log.Printf("Event type: %v\n", meta.EventType)
	log.Printf("Bucket: %v\n", e.Bucket)
	log.Printf("File: %v\n", e.Name)
	log.Printf("Metageneration: %v\n", e.Metageneration)
	log.Printf("Created: %v\n", e.TimeCreated)
	log.Printf("Updated: %v\n", e.Updated)
	uri := fmt.Sprintf("gs://%s/%s", e.Bucket, e.Name)
	log.Printf("URI: %v\n", uri)
	uri = fmt.Sprintf("https://storage.googleapis.com/%s/%s", e.Bucket, e.Name)
	log.Printf("URI: %v\n", uri)

	if strings.HasSuffix(e.Name, ".json") {
		return nil
	}

	client, err := storage.NewClient(context.Background())
	if err != nil {
		return err
	}

	// can we do this?
	obj := client.Bucket(e.Bucket).Object(e.Name)
	log.Printf("obj: %v\n", obj)

	// or this?
	b, err := read(client, e.Bucket, e.Name)
	if err != nil {
		return err
	}
	log.Printf("b: %s\n", b)

	var instance compute.Instance
	err = json.Unmarshal(b, &instance)
	if err != nil {
		return err
	}
	log.Printf("instance: %s\n", b)

	// Check we have an ID since we're very dependent on it
	if instance.Id == 0 {
		return fmt.Errorf("No compute instance ID could be found: %v", instance)
	}

	// Connect to firestore
	fc, err := firestore.NewClient(ctx, firestore.DetectProjectID)
	if err != nil {
		return err
	}

	// Compute a deterministic hash to use as firestore ID
	sha := sha1.New()
	sha.Write([]byte(e.Bucket))
	sha.Write([]byte(fmt.Sprintf("%d", instance.Id)))
	id := hex.EncodeToString(sha.Sum(nil))

	// Map the instance into a doc to be stored
	i := mapInstance(instance)
	i["source"] = e.Bucket

	// Write the doc to the "hosts" collection
	hosts := fc.Collection("hosts")
	result, err := hosts.Doc(id).Set(context.Background(), i)
	if err != nil {
		return err
	}
	log.Printf("result: %v\n", result)
	return err
}

func mapInstance(instance compute.Instance) map[string]interface{} {
	m := map[string]interface{}{
		"name":      fmt.Sprintf("%d", instance.Id),
		"since":     instance.CreationTimestamp,
		"state":     instance.Status,
		"public_ip": instance.NetworkInterfaces[0].AccessConfigs[0].NatIP,
	}
	return m
}

// read is taken from here
// https://github.com/GoogleCloudPlatform/golang-samples/blob/master/storage/objects/main.go
func read(client *storage.Client, bucket, object string) ([]byte, error) {
	ctx := context.Background()
	// [START download_file]
	rc, err := client.Bucket(bucket).Object(object).NewReader(ctx)
	if err != nil {
		return nil, err
	}
	defer rc.Close()

	data, err := ioutil.ReadAll(rc)
	if err != nil {
		return nil, err
	}
	return data, nil
	// [END download_file]
}
