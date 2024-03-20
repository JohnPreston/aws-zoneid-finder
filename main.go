package main

import (
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"io/ioutil"
	"log"
	"net/http"
	"strings"

	"os"
)

func subnetFromEC2() (string, error) {
	// Create an EC2Metadata client
	sess, err := session.NewSession()
	client := ec2metadata.New(sess)

	// Check if the code is running on an EC2 instance
	if !client.Available() {
		return "", fmt.Errorf("Not running on an EC2 instance")
	}
	macAddress, err := client.GetMetadata("network/interfaces/macs/")
	// Retrieve the instance's subnet ID
	subnetID, err := client.GetMetadata(fmt.Sprintf("network/interfaces/macs/%s/subnet-id", macAddress))
	if err != nil {
		log.Fatal(err)
	}
	return subnetID, nil
}

func getSubnetZoneID(subnetID string) (string, error) {
	// Split the subnet ID to extract the availability zone
	parts := strings.Split(subnetID, "-")

	if len(parts) != 3 {
		return "", fmt.Errorf("Invalid subnet ID format: %s", subnetID)
	}
	availabilityZone := parts[2]
	// Extract the last character of the availability zone
	zoneID := string(availabilityZone[len(availabilityZone)-1])

	return zoneID, nil
}

func getEcsMetadatUrl() (string, error) {
	envVarName := "ECS_CONTAINER_METADATA_URI_V4"
	envVarValue := os.Getenv(envVarName)
	return string(envVarValue), nil
}

func getJSONFromURL(url string, target interface{}) error {
	// Send an HTTP GET request to the URL
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	// Parse the JSON response into the target interface
	err = json.Unmarshal(body, target)
	if err != nil {
		return err
	}
	return nil
}

// FindZoneIDByCIDR returns the AWS Subnet Zone ID based on the subnet CIDR.
func FindZoneIDByCIDR(subnetCIDR string) (string, error) {
	// Create an AWS session
	sess, err := session.NewSession()

	// Create an EC2 client
	svc := ec2.New(sess)

	// Describe the subnet
	input := &ec2.DescribeSubnetsInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("cidr"),
				Values: []*string{aws.String(subnetCIDR)},
			},
		},
	}

	result, err := svc.DescribeSubnets(input)
	if err != nil {
		return "", err
	}

	// Check if a subnet was found
	if len(result.Subnets) == 0 {
		return "", fmt.Errorf("Subnet with CIDR %s not found", subnetCIDR)
	}

	// Extract the availability zone from the subnet description
	zoneID := *result.Subnets[0].AvailabilityZoneId

	return zoneID, nil
}

type ContainerInfo struct {
	DockerId string `json:"DockerId"`
	Name     string `json:"Name"`
	// Add other fields as needed
	Networks []struct {
		IPv4SubnetCIDRBlock string `json:"IPv4SubnetCIDRBlock"`
	} `json:"Networks"`
}

func main() {
	ecsUrl, err := getEcsMetadatUrl()
	if err == nil && ecsUrl != "" {
		var data *ContainerInfo = &ContainerInfo{}
		err = getJSONFromURL(ecsUrl, data)
		if err == nil {
			subnetCIDR := data.Networks[0].IPv4SubnetCIDRBlock
			zoneID, err := FindZoneIDByCIDR(subnetCIDR)
			if err == nil {
				fmt.Printf(zoneID)
				os.Exit(0)
			}
		}
	} else {
		subnetID, err := subnetFromEC2()
		if err == nil {
			zoneID, err := getSubnetZoneID(subnetID)
			if err == nil {
				fmt.Printf(zoneID)
				os.Exit(0)
			}
		}
	}
	os.Exit(1)
}
