package inventory

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConvertAWS(t *testing.T) {
	bs := []byte(awsJSON)
	host, err := convertAWS(bs)
	require.NoError(t, err)
	require.Equal(t, "i-0dac5f409ebb47a71", host.Name)
	require.Equal(t, "ec2-13-56-178-141.us-west-1.compute.amazonaws.com", host.Hostname)
	require.Equal(t, "13.56.178.141", host.IP)
	require.Equal(t, "running", host.State)
}

var awsJSON = `{
	"AmiLaunchIndex": 0,
	"Architecture": "x86_64",
	"BlockDeviceMappings": [{
		"DeviceName": "/dev/sda1",
		"Ebs": {
		  "AttachTime": "2018-07-05T16:35:00.000Z",
		  "DeleteOnTermination": true,
		  "Status": "attached",
		  "VolumeId": "vol-0ee42427711e034a9"
		}
	  }],
	"ClientToken": "",
	"CpuOptions": {
	  "CoreCount": 1,
	  "ThreadsPerCore": 1
	},
	"EbsOptimized": false,
	"Hypervisor": "xen",
	"ImageId": "ami-001d2c60",
	"InstanceId": "i-0dac5f409ebb47a71",
	"InstanceType": "t2.micro",
	"KeyName": "djohnsto",
	"LaunchTime": "2018-07-05T16:34:59.000Z",
	"Monitoring": {
	  "State": "disabled"
	},
	"NetworkInterfaces": [{
		"Association": {
		  "IpOwnerId": "amazon",
		  "PublicDnsName": "ec2-13-56-178-141.us-west-1.compute.amazonaws.com",
		  "PublicIp": "13.56.178.141"
		},
		"Attachment": {
		  "AttachTime": "2018-07-05T16:34:59.000Z",
		  "AttachmentId": "eni-attach-94bfd87a",
		  "DeleteOnTermination": true,
		  "DeviceIndex": 0,
		  "Status": "attached"
		},
		"Description": "",
		"Groups": [{
			"GroupId": "sg-b93857c1",
			"GroupName": "launch-wizard-2"
		  }],
		"MacAddress": "02:a9:ca:bc:d2:da",
		"NetworkInterfaceId": "eni-73fcaf52",
		"OwnerId": "689951665833",
		"PrivateDnsName": "ip-172-31-31-90.us-west-1.compute.internal",
		"PrivateIpAddress": "172.31.31.90",
		"PrivateIpAddresses": [{
			"Association": {
			  "IpOwnerId": "amazon",
			  "PublicDnsName": "ec2-13-56-178-141.us-west-1.compute.amazonaws.com",
			  "PublicIp": "13.56.178.141"
			},
			"Primary": true,
			"PrivateDnsName": "ip-172-31-31-90.us-west-1.compute.internal",
			"PrivateIpAddress": "172.31.31.90"
		  }],
		"SourceDestCheck": true,
		"Status": "in-use",
		"SubnetId": "subnet-582f183d",
		"VpcId": "vpc-dc473db9"
	  }],
	"Placement": {
	  "AvailabilityZone": "us-west-1c",
	  "GroupName": "",
	  "Tenancy": "default"
	},
	"PrivateDnsName": "ip-172-31-31-90.us-west-1.compute.internal",
	"PrivateIpAddress": "172.31.31.90",
	"PublicDnsName": "ec2-13-56-178-141.us-west-1.compute.amazonaws.com",
	"PublicIpAddress": "13.56.178.141",
	"RootDeviceName": "/dev/sda1",
	"RootDeviceType": "ebs",
	"SecurityGroups": [{
		"GroupId": "sg-b93857c1",
		"GroupName": "launch-wizard-2"
	  }],
	"SourceDestCheck": true,
	"State": {
	  "Code": 16,
	  "Name": "running"
	},
	"StateTransitionReason": "",
	"SubnetId": "subnet-582f183d",
	"Tags": [{
		"Key": "Name",
		"Value": "Webservers"
	  }],
	"VirtualizationType": "hvm",
	"VpcId": "vpc-dc473db9"
  }`
