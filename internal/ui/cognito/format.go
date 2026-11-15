package cognito

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/arn"
	idTypes "github.com/aws/aws-sdk-go-v2/service/cognitoidentity/types"
	idpTypes "github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"

	"aws-tui/internal/ui/core"
)

// cognitoIdpDomain is the start of the name an identity pool gives a user pool
// it trusts, which is followed by the region and then the pool id, as in
// "cognito-idp.us-east-1.amazonaws.com/us-east-1_aBc123"
const cognitoIdpDomain = "cognito-idp."

// userPoolArnOfProvider is the ARN to open the user pool an identity pool
// trusts, which is empty for other providers. The account isn't in the name
// of the provider, so it isn't in the ARN, which only the pages of the app
// use to find the pool.
func userPoolArnOfProvider(providerName string) string {
	var domain, poolId, found = strings.Cut(providerName, "/")
	if !found || !strings.HasPrefix(domain, cognitoIdpDomain) || len(poolId) == 0 {
		return ""
	}
	var region, _, _ = strings.Cut(strings.TrimPrefix(domain, cognitoIdpDomain), ".")
	return arn.ARN{
		Partition: "aws",
		Service:   "cognito-idp",
		Region:    region,
		Resource:  "userpool/" + poolId,
	}.String()
}

// userAttribute is the value of the user's attribute of the name, such as
// "email"
func userAttribute(user idpTypes.UserType, name string) string {
	for _, attribute := range user.Attributes {
		if aws.ToString(attribute.Name) == name {
			return aws.ToString(attribute.Value)
		}
	}
	return ""
}

// signInWith is what users sign in with: their username, or the attributes
// used in its place
func signInWith(pool idpTypes.UserPoolType) string {
	if len(pool.UsernameAttributes) == 0 {
		return "username"
	}
	var names []string
	for _, attribute := range pool.UsernameAttributes {
		names = append(names, string(attribute))
	}
	return strings.Join(names, ", ")
}

func joinValues[T ~string](values []T) string {
	var names []string
	for _, value := range values {
		names = append(names, string(value))
	}
	return strings.Join(names, ", ")
}

// passwordPolicy is the rows of the password policy of the user pool
func passwordPolicy(policies *idpTypes.UserPoolPolicyType) []core.TableRow {
	if policies == nil || policies.PasswordPolicy == nil {
		return nil
	}
	var policy = policies.PasswordPolicy
	return core.DetailsSection("Password Policy",
		core.TableRow{"Minimum Length", core.FormatOptionalInt(policy.MinimumLength)},
		core.TableRow{"Require Uppercase", strconv.FormatBool(policy.RequireUppercase)},
		core.TableRow{"Require Lowercase", strconv.FormatBool(policy.RequireLowercase)},
		core.TableRow{"Require Numbers", strconv.FormatBool(policy.RequireNumbers)},
		core.TableRow{"Require Symbols", strconv.FormatBool(policy.RequireSymbols)},
		core.TableRow{"Password History", core.FormatOptionalInt(policy.PasswordHistorySize)},
		core.TableRow{"Temporary Password Days", strconv.Itoa(int(policy.TemporaryPasswordValidityDays))},
	)
}

// lambdaTriggers is the rows of the Lambda functions the user pool calls, in
// the order they run in, with the ARN of each
func lambdaTriggers(config *idpTypes.LambdaConfigType) []core.TableRow {
	if config == nil {
		return nil
	}

	var preTokenGeneration = config.PreTokenGeneration
	if version := config.PreTokenGenerationConfig; version != nil && version.LambdaArn != nil {
		preTokenGeneration = version.LambdaArn
	}
	var customEmailSender, customSmsSender *string
	if config.CustomEmailSender != nil {
		customEmailSender = config.CustomEmailSender.LambdaArn
	}
	if config.CustomSMSSender != nil {
		customSmsSender = config.CustomSMSSender.LambdaArn
	}

	var rows []core.TableRow
	for _, trigger := range []struct {
		name string
		arn  *string
	}{
		{"Pre Sign-up", config.PreSignUp},
		{"User Migration", config.UserMigration},
		{"Pre Authentication", config.PreAuthentication},
		{"Define Auth Challenge", config.DefineAuthChallenge},
		{"Create Auth Challenge", config.CreateAuthChallenge},
		{"Verify Auth Challenge", config.VerifyAuthChallengeResponse},
		{"Pre Token Generation", preTokenGeneration},
		{"Post Authentication", config.PostAuthentication},
		{"Post Confirmation", config.PostConfirmation},
		{"Custom Message", config.CustomMessage},
		{"Custom Email Sender", customEmailSender},
		{"Custom SMS Sender", customSmsSender},
	} {
		if trigger.arn != nil {
			rows = append(rows, core.TableRow{trigger.name, aws.ToString(trigger.arn)})
		}
	}
	return core.DetailsSection("Lambda Triggers", rows...)
}

// mappingRule is when the rule gives its role, such as
// "cognito:groups Contains admin"
func mappingRule(rule idTypes.MappingRule) string {
	return fmt.Sprintf("%s %s %s", aws.ToString(rule.Claim), rule.MatchType, aws.ToString(rule.Value))
}

// ambiguousResolution is what is done for a user whose role can't be chosen
// by a mapping
func ambiguousResolution(resolution idTypes.AmbiguousRoleResolutionType) string {
	switch resolution {
	case idTypes.AmbiguousRoleResolutionTypeAuthenticatedRole:
		return "Use the authenticated role"
	case idTypes.AmbiguousRoleResolutionTypeDeny:
		return "Deny"
	}
	return string(resolution)
}
