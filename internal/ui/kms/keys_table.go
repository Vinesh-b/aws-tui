package kms

import (
	"slices"
	"strings"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"
	"aws-tui/internal/utils"

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
	allKeys     []kmsKey
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
		),
		data:       nil,
		allKeys:    nil,
		serviceCtx: serviceContext,
	}

	table.populateKeysTable()
	table.SetSelectionChangedFunc(func(row, column int) {})
	table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey { return event })
	table.SetSelectedFunc(func(row, column int) {})
	table.SetSearchDoneFunc(func(key tcell.Key) {
		switch key {
		case core.APP_KEY_BINDINGS.Done:
			table.RefreshKeys(false)
		}
	})

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
	inst.GetCell(0, 0).SetExpansion(1)
	inst.Select(1, 0)
}

// RefreshKeys loads the keys and their aliases, or only searches the keys that
// are here when there is search text and force is not set.
func (inst *KmsKeysTable) RefreshKeys(force bool) {
	inst.refreshKeys(force, func() {})
}

func (inst *KmsKeysTable) refreshKeys(force bool, loaded func()) {
	var search = inst.GetSearchText()
	var allKeys = inst.allKeys
	var data []kmsKey
	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.App, 10)

	// The keys are only handled on the UI goroutine, so a refresh that is
	// still loading can't change what another one searches
	dataLoader.AsyncLoadData(func() {
		if len(search) == 0 || force || allKeys == nil {
			var keys, keysErr = inst.serviceCtx.Api.ListKeys(inst.serviceCtx.Ctx, force)
			var aliases, aliasesErr = inst.serviceCtx.Api.ListAliases(inst.serviceCtx.Ctx, force)
			allKeys = joinAliases(keys, aliases)
			for _, err := range []error{keysErr, aliasesErr} {
				if err != nil {
					inst.ErrorMessageCallback("%s", err.Error())
				}
			}
		}

		if len(search) > 0 {
			data = utils.FuzzySearch(search, allKeys, func(k kmsKey) string {
				return k.aliasText() + " " + k.keyId
			})
		} else {
			data = allKeys
		}
	})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		inst.allKeys = allKeys
		inst.data = data
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
	var idx = slices.IndexFunc(inst.allKeys, func(k kmsKey) bool {
		return "key/"+k.keyId == resource || slices.Contains(k.aliases, resource)
	})
	if idx < 0 {
		return false
	}

	inst.data = inst.allKeys
	inst.populateKeysTable()
	inst.Select(idx+1, 0)
	return true
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
