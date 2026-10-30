package kms

import (
	"slices"
	"strings"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/kms/types"

	"github.com/gdamore/tcell/v2"
)

// kmsKey is a key with the aliases that point to it
type kmsKey struct {
	keyId   string
	keyArn  string
	aliases []string
}

func (inst kmsKey) aliasText() string {
	return strings.Join(inst.aliases, ", ")
}

// joinAliases gives each key the aliases that point to it
func joinAliases(keys []types.KeyListEntry, aliases []types.AliasListEntry) []kmsKey {
	var byKeyId = map[string][]string{}
	for _, alias := range aliases {
		// Aliases without a target are not in use
		if target := aws.ToString(alias.TargetKeyId); target != "" {
			byKeyId[target] = append(byKeyId[target], aws.ToString(alias.AliasName))
		}
	}

	var result = make([]kmsKey, 0, len(keys))
	for _, key := range keys {
		var keyId = aws.ToString(key.KeyId)
		result = append(result, kmsKey{
			keyId:   keyId,
			keyArn:  aws.ToString(key.KeyArn),
			aliases: byKeyId[keyId],
		})
	}
	return result
}

type KmsKeysTable struct {
	*core.SelectableTable[kmsKey]
	data        []kmsKey
	selectedKey kmsKey
	serviceCtx  *core.ServiceContext[awsapi.KmsApi]
}

func NewKmsKeysTable(
	serviceContext *core.ServiceContext[awsapi.KmsApi],
) *KmsKeysTable {

	var table = &KmsKeysTable{
		SelectableTable: core.NewSelectableTable[kmsKey](
			"KMS Keys",
			core.TableRow{
				"Aliases",
				"Key ID",
			},
			serviceContext.AppContext,
		).SetColumnExpansion(0, 1).SetFilterColumns(0, 1),
		data:       nil,
		serviceCtx: serviceContext,
	}

	table.populateKeysTable()
	table.SetSelectionChangedFunc(func(row, column int) {})
	table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey { return event })
	table.SetSelectedFunc(func(row, column int) {})

	return table
}

func (inst *KmsKeysTable) populateKeysTable() {
	var tableData []core.TableRow
	for _, row := range inst.data {
		tableData = append(tableData, core.TableRow{row.aliasText(), row.keyId})
	}

	// Nothing is selected in an empty table
	if len(inst.data) == 0 {
		inst.selectedKey = kmsKey{}
	}

	inst.SetData(tableData, inst.data, 0)
	inst.Select(1, 0)
}

// RefreshKeys loads the keys and their aliases
func (inst *KmsKeysTable) RefreshKeys(force bool) {
	inst.refreshKeys(force, func() {})
}

func (inst *KmsKeysTable) refreshKeys(force bool, loaded func()) {
	var keys []kmsKey
	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.App, 10)

	dataLoader.AsyncLoadData(func() {
		var listed, keysErr = inst.serviceCtx.Api.ListKeys(inst.serviceCtx.Ctx, force)
		var aliases, aliasesErr = inst.serviceCtx.Api.ListAliases(inst.serviceCtx.Ctx, force)
		keys = joinAliases(listed, aliases)
		for _, err := range []error{keysErr, aliasesErr} {
			if err != nil {
				inst.ErrorMessageCallback("%s", err.Error())
			}
		}
	})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		inst.data = keys
		inst.populateKeysTable()
		loaded()
	})
}

// Open selects the key of a "key/<id>" or "alias/<name>" ARN resource. The
// keys are loaded again first if it isn't one of those loaded.
func (inst *KmsKeysTable) Open(resource string) {
	if inst.selectResource(resource) {
		return
	}

	inst.refreshKeys(true, func() {
		if !inst.selectResource(resource) {
			inst.ErrorMessageCallback("Key %s not found", resource)
		}
	})
}

// selectResource shows all the keys, without a search, to select the one the
// resource is for.
func (inst *KmsKeysTable) selectResource(resource string) bool {
	var matches = func(k kmsKey) bool {
		return "key/"+k.keyId == resource || slices.Contains(k.aliases, resource)
	}
	if !slices.ContainsFunc(inst.data, matches) {
		return false
	}

	inst.ClearFilter()
	inst.populateKeysTable()
	for row := 1; row < inst.GetTable().GetRowCount(); row++ {
		if matches(inst.GetPrivateData(row, 0)) {
			inst.Select(row, 0)
			return true
		}
	}
	return false
}

func (inst *KmsKeysTable) SetSelectionChangedFunc(handler func(row int, column int)) {
	inst.SelectableTable.SetSelectionChangedFunc(func(row, column int) {
		if row < 1 {
			return
		}
		inst.selectedKey = inst.GetPrivateData(row, 0)
		handler(row, column)
	})
}

func (inst *KmsKeysTable) SetInputCapture(capture func(event *tcell.EventKey) *tcell.EventKey) {
	inst.SelectableTable.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case core.APP_KEY_BINDINGS.Reset:
			inst.RefreshKeys(true)
			return nil
		}
		return capture(event)
	})
}

func (inst *KmsKeysTable) SetSelectedFunc(handler func(row, column int)) {
	inst.SelectableTable.SetSelectedFunc(func(row, column int) {
		if row < 1 {
			return
		}
		inst.selectedKey = inst.GetPrivateData(row, 0)
		handler(row, column)
	})
}

func (inst *KmsKeysTable) GetSelectedKey() kmsKey {
	return inst.selectedKey
}
