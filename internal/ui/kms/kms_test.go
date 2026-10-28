package kms

import (
	"errors"
	"testing"
	"time"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"
	"aws-tui/internal/ui/kms/kmsfake"
	"aws-tui/internal/ui/uitest"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/kms/types"
)

var keyHeadings = core.TableRow{"Aliases", "Key ID"}

func newKeyEntry(keyId string) types.KeyListEntry {
	return types.KeyListEntry{KeyId: aws.String(keyId), KeyArn: aws.String("arn:aws:kms:us-east-1:123456789012:key/" + keyId)}
}

func newAlias(name string, keyId string) types.AliasListEntry {
	return types.AliasListEntry{AliasName: aws.String(name), TargetKeyId: aws.String(keyId)}
}

func newMetadata(keyId string) *types.KeyMetadata {
	var created = time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)
	return &types.KeyMetadata{
		KeyId:                aws.String(keyId),
		Arn:                  aws.String("arn:aws:kms:us-east-1:123456789012:key/" + keyId),
		Description:          aws.String("Orders"),
		KeyState:             types.KeyStateEnabled,
		Enabled:              true,
		KeyManager:           types.KeyManagerTypeCustomer,
		KeyUsage:             types.KeyUsageTypeEncryptDecrypt,
		KeySpec:              types.KeySpecSymmetricDefault,
		Origin:               types.OriginTypeAwsKms,
		CreationDate:         &created,
		EncryptionAlgorithms: []types.EncryptionAlgorithmSpec{types.EncryptionAlgorithmSpecSymmetricDefault},
	}
}

func newApi() *kmsfake.Api {
	return kmsfake.New(
		kmsfake.WithKeys([]types.KeyListEntry{newKeyEntry("1111"), newKeyEntry("2222")}),
		kmsfake.WithAliases([]types.AliasListEntry{
			newAlias("alias/orders", "1111"), newAlias("alias/orders-old", "1111"), newAlias("alias/unused", ""),
		}),
		kmsfake.WithMetadata("1111", newMetadata("1111")),
		kmsfake.WithTags([]types.Tag{{TagKey: aws.String("team"), TagValue: aws.String("payments")}}),
		kmsfake.WithPolicy(`{"Version":"2012-10-17"}`),
	)
}

func TestJoinAliases__GivesEachKeyItsAliases(t *testing.T) {
	var keys = joinAliases(
		[]types.KeyListEntry{newKeyEntry("1111"), newKeyEntry("2222")},
		[]types.AliasListEntry{newAlias("alias/a", "1111"), newAlias("alias/b", "1111"), newAlias("alias/none", "")},
	)

	if got := keys[0].aliasText(); got != "alias/a, alias/b" {
		t.Errorf("aliases of the first key = %q", got)
	}
	if got := keys[1].aliasText(); got != "" {
		t.Errorf("aliases of the second key = %q, want none", got)
	}
}

func TestKeysTable__RefreshLoadsKeysWithAliases(t *testing.T) {
	var serviceCtx = uitest.NewServiceContextWith[awsapi.KmsApi](newApi())
	var table = NewKmsKeysTable(serviceCtx)
	uitest.RunApp(t, serviceCtx.App)

	table.RefreshKeys(true)

	uitest.AwaitRows(t, serviceCtx.App, table, []core.TableRow{
		keyHeadings,
		{"alias/orders, alias/orders-old", "1111"},
		{"", "2222"},
	})
}

func TestKeysTable__RefreshReportsApiError(t *testing.T) {
	var api = kmsfake.New(kmsfake.WithErr(errors.New("access denied")))
	var serviceCtx = uitest.NewServiceContextWith[awsapi.KmsApi](api)
	var table = NewKmsKeysTable(serviceCtx)
	uitest.RunApp(t, serviceCtx.App)
	var callback, received = uitest.CaptureErrors()
	table.ErrorMessageCallback = callback

	table.RefreshKeys(true)

	if got := uitest.AwaitError(t, received); got != "access denied" {
		t.Errorf("message = %q, want %q", got, "access denied")
	}
}

func TestKeysTable__SearchFiltersTheKeysWithoutListingThemAgain(t *testing.T) {
	var api = newApi()
	var serviceCtx = uitest.NewServiceContextWith[awsapi.KmsApi](api)
	var table = NewKmsKeysTable(serviceCtx)
	uitest.RunApp(t, serviceCtx.App)
	table.RefreshKeys(true)
	uitest.Eventually(t, serviceCtx.App, func() bool { return table.GetTable().GetRowCount() == 3 })
	// Setting the search text searches the keys, as when the search is done
	uitest.OnUI(t, serviceCtx.App, func() { table.SetSearchText("2222") })

	uitest.AwaitRows(t, serviceCtx.App, table, []core.TableRow{keyHeadings, {"", "2222"}})
	if got := api.KeyListings(); got != 1 {
		t.Errorf("keys were listed %d times, want 1", got)
	}
}

func TestKeysTable__SelectingRowTracksSelectedKey(t *testing.T) {
	var table = NewKmsKeysTable(uitest.NewServiceContext[awsapi.KmsApi]())
	table.data = joinAliases([]types.KeyListEntry{newKeyEntry("1111"), newKeyEntry("2222")}, nil)
	table.populateKeysTable()

	table.Select(2, 0)

	if got := table.GetSelectedKey().keyId; got != "2222" {
		t.Errorf("selected key = %q, want 2222", got)
	}
}

func TestKeysTable__EmptyTableHasNoSelectedKey(t *testing.T) {
	var table = NewKmsKeysTable(uitest.NewServiceContext[awsapi.KmsApi]())
	table.data = joinAliases([]types.KeyListEntry{newKeyEntry("1111")}, nil)
	table.populateKeysTable()
	table.Select(1, 0)

	table.data = nil
	table.populateKeysTable()

	if got := table.GetSelectedKey().keyId; got != "" {
		t.Errorf("selected key = %q, want none", got)
	}
}

func TestKeyDetailsTable__PopulatesFromKeyMetadata(t *testing.T) {
	var table = NewKmsKeyDetailsTable(uitest.NewServiceContext[awsapi.KmsApi]())
	table.selectedKey = kmsKey{keyId: "1111", aliases: []string{"alias/orders"}}
	table.data = newMetadata("1111")

	table.populateDetailsTable()

	uitest.AssertDetailsRows(t, table.DetailsTable, []core.TableRow{
		{"Key ID", "1111"},
		{"Aliases", "alias/orders"},
		{"Description", "Orders"},
		{"State", "Enabled"},
		{"Enabled", "true"},
		{"Key Manager", "CUSTOMER"},
		{"Usage", "ENCRYPT_DECRYPT"},
		{"Key Spec", "SYMMETRIC_DEFAULT"},
		{"Origin", "AWS_KMS"},
		{"Multi-Region", "false"},
		{"Created", "2024-01-02 03:04:05"},
		{"Scheduled Deletion", ""},
		{"ARN", "arn:aws:kms:us-east-1:123456789012:key/1111"},
		{"Algorithms", ""},
		{"Encryption", "SYMMETRIC_DEFAULT"},
		{"Signing", ""},
	})
}

func TestKeyDetailsTable__RefreshLoadsDescriptionAndTags(t *testing.T) {
	var serviceCtx = uitest.NewServiceContextWith[awsapi.KmsApi](newApi())
	var table = NewKmsKeyDetailsTable(serviceCtx)
	uitest.RunApp(t, serviceCtx.App)
	table.SetSelectedKey(kmsKey{keyId: "1111"})

	table.RefreshDetails()

	uitest.AwaitDetailsTag(t, serviceCtx.App, table.DetailsTable, "team", "payments")
}
