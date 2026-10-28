package ec2

import (
	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

// Ec2DetailsTable shows the selected instance. Everything it shows, tags
// included, comes with the instance in the list, so nothing is loaded.
type Ec2DetailsTable struct {
	*core.DetailsTable
	data *types.Instance
}

func NewEc2DetailsTable(
	serviceCtx *core.ServiceContext[awsapi.Ec2Api],
) *Ec2DetailsTable {
	var table = &Ec2DetailsTable{
		DetailsTable: core.NewDetailsTable("Instance Details", serviceCtx.AppContext),
	}

	table.populateDetailsTable()

	return table
}

func instanceRows(instance types.Instance) []core.TableRow {
	var zone, tenancy string
	if instance.Placement != nil {
		zone = aws.ToString(instance.Placement.AvailabilityZone)
		tenancy = string(instance.Placement.Tenancy)
	}
	var monitoring string
	if instance.Monitoring != nil {
		monitoring = string(instance.Monitoring.State)
	}
	var profile string
	if instance.IamInstanceProfile != nil {
		profile = aws.ToString(instance.IamInstanceProfile.Arn)
	}

	var rows = []core.TableRow{
		{"Name", instanceName(instance)},
		{"Instance ID", aws.ToString(instance.InstanceId)},
		{"State", instanceState(instance)},
		{"Type", string(instance.InstanceType)},
		{"Launched", core.FormatTime(instance.LaunchTime)},
		{"Availability Zone", zone},
		{"Tenancy", tenancy},
		{"Architecture", string(instance.Architecture)},
		{"Platform", aws.ToString(instance.PlatformDetails)},
		{"AMI", aws.ToString(instance.ImageId)},
		{"Key Pair", aws.ToString(instance.KeyName)},
		{"IAM Instance Profile", profile},
		{"Monitoring", monitoring},
	}

	rows = append(rows, core.DetailsSection("Network",
		core.TableRow{"VPC ID", aws.ToString(instance.VpcId)},
		core.TableRow{"Subnet ID", aws.ToString(instance.SubnetId)},
		core.TableRow{"Private IP", aws.ToString(instance.PrivateIpAddress)},
		core.TableRow{"Private DNS", aws.ToString(instance.PrivateDnsName)},
		core.TableRow{"Public IP", aws.ToString(instance.PublicIpAddress)},
		core.TableRow{"Public DNS", aws.ToString(instance.PublicDnsName)},
	)...)

	var groups []core.TableRow
	for _, group := range instance.SecurityGroups {
		groups = append(groups, core.TableRow{aws.ToString(group.GroupId), aws.ToString(group.GroupName)})
	}
	rows = append(rows, core.DetailsSection("Security Groups", groups...)...)

	var volumes []core.TableRow
	for _, mapping := range instance.BlockDeviceMappings {
		var volumeId, status string
		if mapping.Ebs != nil {
			volumeId = aws.ToString(mapping.Ebs.VolumeId)
			status = string(mapping.Ebs.Status)
		}
		volumes = append(volumes, core.TableRow{aws.ToString(mapping.DeviceName), volumeId + " " + status})
	}
	rows = append(rows, core.DetailsSection("Volumes", volumes...)...)

	var tags = core.TagRows(instance.Tags, func(t types.Tag) (string, string) {
		return aws.ToString(t.Key), aws.ToString(t.Value)
	})
	return append(rows, core.DetailsSection("Tags", tags...)...)
}

func (inst *Ec2DetailsTable) populateDetailsTable() {
	var tableData []core.TableRow
	if inst.data != nil {
		tableData = instanceRows(*inst.data)
		inst.SetTitleExtra(aws.ToString(inst.data.InstanceId))
		inst.SetRawData(*inst.data)
	} else {
		inst.SetTitleExtra("")
		inst.SetRawData(nil)
	}

	inst.SetData(tableData)
	inst.Select(0, 0)
	inst.ScrollToBeginning()
}

// SetInstance shows the instance
func (inst *Ec2DetailsTable) SetInstance(instance types.Instance) {
	inst.data = &instance
	inst.populateDetailsTable()
}
