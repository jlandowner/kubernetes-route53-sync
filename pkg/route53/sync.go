package route53

import (
	"context"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	"github.com/pkg/errors"
)

var (
	// ErrRecordSetNotFound tell us to create New RecordSet
	ErrRecordSetNotFound = errors.New("RecordSet is not found")
)

// Sync update Route53 RecordSet by given ips
func Sync(ctx context.Context, ips []string, dnsNames []string, ttl int64, zoneID string) error {

	cfg, err := external.LoadDefaultAWSConfig()
	if err != nil {
		return errors.Wrap(err, "unable to load SDK config")
	}
	cfg.Region = "us-east-1"
	r53 := route53.New(cfg)

	root := dnsNames[0]
	if zoneID == "" {
		zoneID, err = findHostedZoneID(ctx, r53.ListHostedZonesRequest(&route53.ListHostedZonesInput{}), root)
		if err != nil {
			return errors.Wrapf(err, "failed to find zone id for dns-name:=%s", root)
		}
	}

	for _, dnsName := range dnsNames {
		// get current ResourceRecordSet
		recordSet, err := getResourceRecordSet(ctx,
			r53.ListResourceRecordSetsRequest(&route53.ListResourceRecordSetsInput{HostedZoneId: &zoneID}), dnsName)
		if err != nil {
			// if not found, create new ResourceRecordSet
			if err == ErrRecordSetNotFound {
				recordSet = &route53.ResourceRecordSet{
					Name: &dnsName,
					Type: route53.RRTypeA,
					TTL:  &ttl,
				}
			} else {
				return errors.Wrapf(err, "failed to get dns records for zone-id=%s name=%s",
					zoneID, dnsName)
			}
		}

		// change ResourceRecords to given ips
		records := make([]route53.ResourceRecord, 0)
		for _, ip := range ips {
			value := ip
			records = append(records, route53.ResourceRecord{Value: &value})
		}
		recordSet.ResourceRecords = records
		recordSet.TTL = &ttl

		// upsert Route53 RecordSets
		if err := upsertResourceRecordSets(ctx, r53, zoneID, recordSet); err != nil {
			return errors.Wrapf(err, "failed to upsert dns records for zone-id=%s name=%s",
				zoneID, dnsName)
		}
	}
	return nil
}

// findHostedZoneID finds a hostedzone id for the given dns record
func findHostedZoneID(ctx context.Context, sender interface {
	Send(ctx context.Context) (*route53.ListHostedZonesResponse, error)
}, dnsName string) (string, error) {
	result, err := sender.Send(ctx)
	if err != nil {
		return "", err
	}

	for _, hostedZone := range result.HostedZones {
		if *hostedZone.Name == dnsName || strings.HasSuffix(dnsName, "."+*hostedZone.Name) {
			return *hostedZone.Id, nil
		}
	}
	return "", errors.New("HostedZone id not found")
}

// getResourceRecordSet returns ResourceRecordSet when it is found and is A type record
func getResourceRecordSet(ctx context.Context, sender interface {
	Send(ctx context.Context) (*route53.ListResourceRecordSetsResponse, error)
}, recordName string) (*route53.ResourceRecordSet, error) {
	result, err := sender.Send(ctx)
	if err != nil {
		return nil, err
	}

	for _, recordSet := range result.ResourceRecordSets {
		if *recordSet.Name == recordName {
			if recordSet.Type == route53.RRTypeA {
				return &recordSet, nil
			}
			return nil, errors.New("RecordSet is found but NOT A record")
		}
	}
	return nil, ErrRecordSetNotFound
}

// upsertResourceRecordSets create or update a Route53 record
func upsertResourceRecordSets(
	ctx context.Context, r53 *route53.Client, hostedzoneID string, recordSet *route53.ResourceRecordSet) error {
	comment := "kubernetes-route53-sync update"
	batch := &route53.ChangeBatch{
		Changes: []route53.Change{
			{
				Action:            route53.ChangeActionUpsert,
				ResourceRecordSet: recordSet,
			},
		},
		Comment: &comment,
	}
	res, err := r53.ChangeResourceRecordSetsRequest(
		&route53.ChangeResourceRecordSetsInput{HostedZoneId: &hostedzoneID, ChangeBatch: batch}).Send(ctx)

	log.Printf("Route53 Change Info %v\n", res)
	return err
}
