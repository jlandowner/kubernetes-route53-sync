package route53

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

const (
	existID            = "/hostedzone/EXAMPLECOMCOM"
	notExistID         = "/hostedzone/XXXXXXXXXXXXX"
	existHZName        = "example.com."
	existDNSName       = "test.example.com."
	notExistDNSName    = "notexist.example.com."
	notExistDNSName2   = "notexist.notexist.com."
	secondExistDNSName = "test2.example.com."
)

type MockRoute53Client struct{}

func (m *MockRoute53Client) ListHostedZonesRequest(input *route53.ListHostedZonesInput) *MockListHostedZonesRequest {
	return &MockListHostedZonesRequest{}
}

type MockListHostedZonesRequest struct{}

func (m *MockListHostedZonesRequest) Send(ctx context.Context) (*route53.ListHostedZonesResponse, error) {
	name := "example.com."
	id := "/hostedzone/EXAMPLECOMCOM"

	res := &route53.ListHostedZonesResponse{}
	res.ListHostedZonesOutput = &route53.ListHostedZonesOutput{
		HostedZones: []route53.HostedZone{
			{Name: &name, Id: &id},
		},
	}
	return res, nil
}

func (m *MockRoute53Client) ListResourceRecordSetsRequest(input *route53.ListResourceRecordSetsInput) *MockListResourceRecordSetsRequest {
	return &MockListResourceRecordSetsRequest{
		HostedZoneID: input.HostedZoneId,
	}
}

type MockListResourceRecordSetsRequest struct {
	HostedZoneID *string
}

func (m *MockListResourceRecordSetsRequest) Send(ctx context.Context) (*route53.ListResourceRecordSetsResponse, error) {
	name := "example.com."
	name1 := "test.example.com."
	name2 := "test2.example.com."

	nsrecords := []string{"ns-774.awsdns-32.net.", "ns-30.awsdns-03.com.", "ns-2000.awsdns-58.co.uk.", "ns-1241.awsdns-27.org."}
	soarecord := "ns-774.awsdns-32.net. awsdns-hostmaster.amazon.com."
	arecords := []string{"1.1.1.1", "2.2.2.2", "3.3.3.3"}
	ttl := int64(60)

	if *m.HostedZoneID == notExistID {
		return nil, errors.New("HostedZone is not found")
	}
	res := &route53.ListResourceRecordSetsResponse{}
	res.ListResourceRecordSetsOutput = &route53.ListResourceRecordSetsOutput{
		ResourceRecordSets: []route53.ResourceRecordSet{
			{Name: &name, Type: route53.RRTypeNs, TTL: &ttl, ResourceRecords: []route53.ResourceRecord{
				{Value: &nsrecords[0]}, {Value: &nsrecords[1]},
				{Value: &nsrecords[2]}, {Value: &nsrecords[3]},
			}},
			{Name: &name, Type: route53.RRTypeSoa, TTL: &ttl, ResourceRecords: []route53.ResourceRecord{
				{Value: &soarecord},
			}},
			{Name: &name1, Type: route53.RRTypeA, TTL: &ttl, ResourceRecords: []route53.ResourceRecord{
				{Value: &arecords[0]}, {Value: &arecords[1]}, {Value: &arecords[2]},
			}},
			{Name: &name2, Type: route53.RRTypeCname, TTL: &ttl, ResourceRecords: []route53.ResourceRecord{
				{Value: &name1},
			}},
		},
	}
	return res, nil
}

func TestFindHostedZoneID(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*3000)
	defer cancel()
	// r53 := createRoute53Client()
	r53 := MockRoute53Client{}

	t.Run("Exist", func(t *testing.T) {
		expectedID := existID
		dnsName := existDNSName

		id, err := findHostedZoneID(ctx, r53.ListHostedZonesRequest(&route53.ListHostedZonesInput{}), dnsName)
		assert.Nil(t, err)
		assert.Equal(t, expectedID, id)
	})

	t.Run("Not Exist", func(t *testing.T) {
		dnsName := notExistDNSName2
		_, err := findHostedZoneID(ctx, r53.ListHostedZonesRequest(&route53.ListHostedZonesInput{}), dnsName)
		assert.NotNil(t, err)
	})
}

func TestGetResourceRecordSet(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*3000)
	defer cancel()
	r53 := MockRoute53Client{}

	t.Run("Exist", func(t *testing.T) {
		recordName := existDNSName
		zoneID := existID
		ips := []string{"1.1.1.1", "2.2.2.2", "3.3.3.3"}
		ttl := int64(60)
		expectedRecordSet := &route53.ResourceRecordSet{
			Name: &recordName,
			ResourceRecords: []route53.ResourceRecord{
				{Value: &ips[0]}, {Value: &ips[1]}, {Value: &ips[2]},
			},
			TTL:  &ttl,
			Type: route53.RRTypeA,
		}

		recordSet, err := getResourceRecordSet(ctx,
			r53.ListResourceRecordSetsRequest(&route53.ListResourceRecordSetsInput{HostedZoneId: &zoneID}),
			recordName)
		assert.Nil(t, err)
		assert.Equal(t, recordSet, expectedRecordSet)
	})
	t.Run("Not Exist", func(t *testing.T) {
		recordName := notExistDNSName
		zoneID := existID

		_, err := getResourceRecordSet(ctx,
			r53.ListResourceRecordSetsRequest(&route53.ListResourceRecordSetsInput{HostedZoneId: &zoneID}), recordName)
		assert.NotNil(t, err)
	})

}

func createRoute53Client(useMock bool) *route53.Client {
	os.Setenv("AWS_ACCESS_KEY_ID", "<YOUR_AWS_ACCESS_KEY_ID>")
	os.Setenv("AWS_SECRET_ACCESS_KEY", "<YOUR_AWS_SECRET_ACCESS_KEY>")

	cfg, err := external.LoadDefaultAWSConfig()
	if err != nil {
		panic(errors.Wrap(err, "unable to load SDK config"))
	}

	r53 := route53.New(cfg)
	return r53
}
