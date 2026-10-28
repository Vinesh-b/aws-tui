package secretsmanager

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"aws-tui/internal/awsapi"
	"aws-tui/internal/ui/core"

	"github.com/aws/aws-sdk-go-v2/aws"
)

// SecretValueView shows the value of a secret. The value is only fetched when
// it is asked for, so that it is not on the screen just from scrolling through
// the secrets.
type SecretValueView struct {
	*core.SearchableTextView
	ErrorMessageCallback func(text string, a ...any)
	serviceCtx           *core.ServiceContext[awsapi.SecretsManagerApi]
}

func NewSecretValueView(
	serviceCtx *core.ServiceContext[awsapi.SecretsManagerApi],
) *SecretValueView {
	var view = &SecretValueView{
		SearchableTextView:   core.NewSearchableTextView("Value", serviceCtx.AppContext),
		ErrorMessageCallback: func(text string, a ...any) {},
		serviceCtx:           serviceCtx,
	}
	view.SetJsonHighlighting(true)

	return view
}

// Clear removes the value, which belongs to the secret it was fetched for
func (inst *SecretValueView) Clear() {
	inst.SetText("", false)
}

// RefreshValue fetches the value of the secret and calls onLoaded once it is
// shown
func (inst *SecretValueView) RefreshValue(secretId string, onLoaded func()) {
	var dataLoader = core.NewUiDataLoader(inst.serviceCtx.App, 10)
	var value string
	var failed bool

	dataLoader.AsyncLoadData(func() {
		var output, err = inst.serviceCtx.Api.GetSecretValue(inst.serviceCtx.Ctx, secretId)
		if err != nil {
			failed = true
			inst.ErrorMessageCallback("%s", err.Error())
			return
		}

		value = aws.ToString(output.SecretString)
		if output.SecretString == nil && output.SecretBinary != nil {
			value = fmt.Sprintf("<binary secret, %d bytes>", len(output.SecretBinary))
		}
	})

	dataLoader.AsyncUpdateView(inst.Box, func() {
		if failed {
			return
		}
		inst.SetText(indentJson(value), false)
		onLoaded()
	})
}

// indentJson lays out a json value over lines and leaves any other text as it is
func indentJson(text string) string {
	var indented bytes.Buffer
	if err := json.Indent(&indented, []byte(text), "", "  "); err != nil {
		return text
	}
	return indented.String()
}

// isJsonStructure is whether the text is a json object or array
func isJsonStructure(text string) bool {
	var trimmed = strings.TrimSpace(text)
	return json.Valid([]byte(trimmed)) && (strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "["))
}
