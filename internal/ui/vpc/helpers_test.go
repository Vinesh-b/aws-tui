package vpc

import "github.com/aws/aws-sdk-go-v2/service/ec2/types"

func nameTag(name string) []types.Tag {
	var key, value = "Name", name
	return []types.Tag{{Key: &key, Value: &value}}
}
