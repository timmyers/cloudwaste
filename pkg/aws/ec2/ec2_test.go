package ec2

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockedEC2 struct {
	mock.Mock
	ec2iface.EC2API
}

func (m *mockedEC2) DescribeAddressesWithContext(ctx context.Context, input *ec2.DescribeAddressesInput, options ...request.Option) (*ec2.DescribeAddressesOutput, error) {
	args := m.Called(ctx, input, options)

	return args.Get(0).(*ec2.DescribeAddressesOutput), args.Error(1)
}

func (m *mockedEC2) DescribeNatGatewaysPagesWithContext(ctx context.Context, input *ec2.DescribeNatGatewaysInput, fn func(*ec2.DescribeNatGatewaysOutput, bool) bool, opts ...request.Option) error {
	args := m.Called(ctx, input, fn)

	fn(args.Get(0).(*ec2.DescribeNatGatewaysOutput), true)
	return args.Error(1)
}

func (m *mockedEC2) DescribeRouteTablesWithContext(ctx context.Context, input *ec2.DescribeRouteTablesInput, options ...request.Option) (*ec2.DescribeRouteTablesOutput, error) {
	args := m.Called(ctx, input, options)

	return args.Get(0).(*ec2.DescribeRouteTablesOutput), args.Error(1)
}

func (m *mockedEC2) DescribeVolumesPagesWithContext(ctx context.Context, input *ec2.DescribeVolumesInput, fn func(*ec2.DescribeVolumesOutput, bool) bool, opts ...request.Option) error {
	args := m.Called(ctx, input, fn)

	fn(args.Get(0).(*ec2.DescribeVolumesOutput), true)
	return args.Error(1)
}

func TestGetUnusedElasticIPAddresses(t *testing.T) {
	assert := assert.New(t)

	const alloc2 = "allocation2"

	m := new(mockedEC2)
	m.On("DescribeAddressesWithContext", mock.Anything, mock.Anything, mock.Anything).
		Return(&ec2.DescribeAddressesOutput{
			Addresses: []*ec2.Address{
				{ // used
					AllocationId:  aws.String("allocation1"),
					AssociationId: aws.String("association1"),
				},
				{ // unused
					AllocationId:  aws.String("allocation2"),
					AssociationId: nil,
				},
			},
		}, nil).Once()

	client := EC2{Client: m}
	unusedAddresses, err := client.GetUnusedElasticIPAddresses(context.Background())
	assert.Equal(1, len(unusedAddresses))
	assert.Equal(alloc2, unusedAddresses[0].ID())
	assert.Nil(err)

	m.On("DescribeAddressesWithContext", mock.Anything, mock.Anything, mock.Anything).
		Return(&ec2.DescribeAddressesOutput{}, errors.New("AWS Error"))

	unusedAddresses, err = client.GetUnusedElasticIPAddresses(context.Background())
	assert.Nil(unusedAddresses)
	assert.NotNil(err)
}

func TestGetUnusedNATGateways(t *testing.T) {
	assert := assert.New(t)
	m := new(mockedEC2)

	// Unused
	m.On("DescribeNatGatewaysPagesWithContext", mock.Anything, mock.Anything, mock.Anything).
		Return(&ec2.DescribeNatGatewaysOutput{
			NatGateways: []*ec2.NatGateway{
				{
					NatGatewayId: aws.String("gateway1"),
				},
			},
		}, nil).Once()
	m.On("DescribeRouteTablesWithContext", mock.Anything, mock.Anything, mock.Anything).
		Return(&ec2.DescribeRouteTablesOutput{
			RouteTables: []*ec2.RouteTable{},
		}, nil).Once()

	client := EC2{Client: m}
	unusedNatGateways, err := client.GetUnusedNATGateways(context.Background())
	assert.Equal(1, len(unusedNatGateways))
	assert.Nil(err)

	// Used
	m.On("DescribeNatGatewaysPagesWithContext", mock.Anything, mock.Anything, mock.Anything).
		Return(&ec2.DescribeNatGatewaysOutput{
			NatGateways: []*ec2.NatGateway{
				{
					NatGatewayId: aws.String("gateway1"),
				},
			},
		}, nil).Once()
	m.On("DescribeRouteTablesWithContext", mock.Anything, mock.Anything, mock.Anything).
		Return(&ec2.DescribeRouteTablesOutput{
			RouteTables: []*ec2.RouteTable{
				{
					RouteTableId: aws.String("routetable1"),
				},
			},
		}, nil).Once()

	unusedNatGateways, err = client.GetUnusedNATGateways(context.Background())
	assert.Equal(0, len(unusedNatGateways))
	assert.Nil(err)
}

func TestGetUnusedVolumes(t *testing.T) {
	assert := assert.New(t)
	m := new(mockedEC2)

	const vol1Name = "vol1"

	m.On("DescribeVolumesPagesWithContext", mock.Anything, mock.Anything, mock.Anything).
		Return(&ec2.DescribeVolumesOutput{
			Volumes: []*ec2.Volume{
				{ // used
					VolumeId: aws.String(vol1Name),
					State:    aws.String("available"),
				},
				{ // unused
					VolumeId: aws.String("vol2"),
					State:    aws.String("in-use"),
				},
			},
		}, nil).Once()

	client := EC2{Client: m}
	unusedVolumes, err := client.GetUnusedEBSVolumes(context.Background())

	assert.Equal(1, len(unusedVolumes))
	assert.Equal(vol1Name, unusedVolumes[0].R.ID())
	assert.Nil(err)
}
