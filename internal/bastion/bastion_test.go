package bastion

import (
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"

	"github.com/eichemberger/burrow/internal/configstore"
)

func TestMergeInstancesFromSSM(t *testing.T) {
	ssmIDs := []string{"i-match", "i-filtered", "i-missing"}
	idToInstance := map[string]Instance{
		"i-match": {ID: "i-match", PrivateIP: "10.0.1.1", Name: "bastion"},
	}

	filtered := mergeSSMInstances(ssmIDs, idToInstance, &configstore.EC2Selector{
		TagFilters: []configstore.TagFilter{{Key: "Role", Value: "bastion"}},
	})
	if len(filtered) != 1 {
		t.Fatalf("expected 1 instance with tag filter, got %d", len(filtered))
	}
	if filtered[0].ID != "i-match" {
		t.Fatalf("unexpected instance: %+v", filtered[0])
	}

	unfiltered := mergeSSMInstances(ssmIDs, idToInstance, nil)
	if len(unfiltered) != 3 {
		t.Fatalf("expected 3 instances without tag filter, got %d", len(unfiltered))
	}
	if unfiltered[2].PrivateIP != "" {
		t.Fatalf("expected stub for missing instance, got %+v", unfiltered[2])
	}
}

func TestMatchesAllTagFilters(t *testing.T) {
	tags := []ec2types.Tag{
		{Key: aws.String("Role"), Value: aws.String("bastion")},
		{Key: aws.String("Environment"), Value: aws.String("production")},
	}

	filters := []configstore.TagFilter{
		{Key: "Role", Value: "bastion"},
		{Key: "Environment", Value: "production"},
	}

	if !matchesAllTagFilters(tags, filters) {
		t.Fatal("expected all filters to match")
	}

	partial := []configstore.TagFilter{
		{Key: "Role", Value: "bastion"},
		{Key: "Environment", Value: "staging"},
	}
	if matchesAllTagFilters(tags, partial) {
		t.Fatal("expected partial mismatch to fail")
	}
}

func TestMatchesAllTagFiltersEmptyTags(t *testing.T) {
	filters := []configstore.TagFilter{{Key: "Role", Value: "bastion"}}
	if matchesAllTagFilters(nil, filters) {
		t.Fatal("expected empty tags to fail filter")
	}
}

func TestFinalizeListResultNoSSMInstances(t *testing.T) {
	_, err := finalizeListResult(nil, 0, nil)
	if !errors.Is(err, ErrNoSSMInstances) {
		t.Fatalf("expected ErrNoSSMInstances, got %v", err)
	}
	if errors.Is(err, ErrNoMatchingTagFilters) {
		t.Fatal("did not expect tag filter error")
	}
}

func TestFinalizeListResultNoMatchingTagFilters(t *testing.T) {
	filter := &configstore.EC2Selector{
		TagFilters: []configstore.TagFilter{{Key: "Role", Value: "bastion"}},
	}
	_, err := finalizeListResult(nil, 3, filter)
	if !errors.Is(err, ErrNoMatchingTagFilters) {
		t.Fatalf("expected ErrNoMatchingTagFilters, got %v", err)
	}
	if errors.Is(err, ErrNoSSMInstances) {
		t.Fatal("did not expect no-SSM error")
	}
}

func TestNoSSMInstancesErrorIncludesRegion(t *testing.T) {
	err := noSSMInstancesError("eu-west-1")
	if !errors.Is(err, ErrNoSSMInstances) {
		t.Fatalf("expected ErrNoSSMInstances, got %v", err)
	}
	if err.Error() == ErrNoSSMInstances.Error() {
		t.Fatalf("expected region-specific message, got %q", err.Error())
	}
}
