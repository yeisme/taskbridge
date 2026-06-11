package cmd

import (
	"bufio"
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/yeisme/taskbridge/internal/clioutput"
	"github.com/yeisme/taskbridge/internal/provider"
	"github.com/yeisme/taskbridge/internal/provider/feishu"
	"github.com/yeisme/taskbridge/internal/provider/google"
	"github.com/yeisme/taskbridge/internal/provider/microsoft"
	"github.com/yeisme/taskbridge/internal/provider/ticktick"
	"github.com/yeisme/taskbridge/internal/provider/todoist"
	"github.com/yeisme/taskbridge/pkg/paths"
	"github.com/yeisme/taskbridge/pkg/tokenstore"
	"github.com/yeisme/taskbridge/pkg/ui"
)

// authCmd authentication command
var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Authentication management",
	Long: `Manage Todo provider authentication state.

Supported providers:
  - google: Google Tasks API
  - microsoft: Microsoft To Do
  - feishu: Feishu Tasks
  - ticktick: TickTick
  - dida: Dida365
  - todoist: Todoist

Subcommands:
  login <provider>    Log in to the specified provider
  logout <provider>   Log out of the specified provider
  status              View the authentication status of all providers
  show <provider>     View authentication details for a single provider
  refresh <provider>  Refresh the token of the specified provider

Examples:
  taskbridge auth login google
  taskbridge auth status
  taskbridge auth show ms
  taskbridge auth logout google`,
}

// authLoginCmd login command
var authLoginCmd = &cobra.Command{
	Use:   "login <provider>",
	Short: "Log in to the specified Provider",
	Long: `Log in to the specified Todo Provider for OAuth2 authentication.

Supported Providers:
  - google: Google Tasks API
  - microsoft: Microsoft To Do
  - feishu: Feishu Tasks
  - ticktick: TickTick
  - dida: Dida365
  - todoist: Todoist

Example:
  taskbridge auth login google
  taskbridge auth login google --manual # Manually enter the authorization code`,
	Args: cobra.ExactArgs(1),
	RunE: runAuthLogin,
}

// authLogoutCmd logout command
var authLogoutCmd = &cobra.Command{
	Use:   "logout <provider>",
	Short: "Log out of the specified Provider",
	Long: `Log out of the specified Todo Provider and delete the locally stored token.

Example:
  taskbridge auth logout google`,
	Args: cobra.ExactArgs(1),
	RunE: runAuthLogout,
}

// authStatusCmd state command
var authStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "View the authentication status of all providers",
	Long: `Display authentication state for every configured provider.

Example:
  taskbridge auth status`,
	RunE: runAuthStatus,
}

// authShowCmd details command
var authShowCmd = &cobra.Command{
	Use:   "show <provider>",
	Short: "View authentication details for a single provider",
	Long: `Display authentication details for the specified provider. Abbreviations are supported.

Examples:
  taskbridge auth show microsoft
  taskbridge auth show ms`,
	Args: cobra.ExactArgs(1),
	RunE: runAuthShow,
}

// authRefreshCmd refresh command
var authRefreshCmd = &cobra.Command{
	Use:   "refresh <provider>",
	Short: "Refresh the token of the specified Provider",
	Long: `Refresh the OAuth2 token for the specified Provider.

Example:
  taskbridge auth refresh google`,
	Args: cobra.ExactArgs(1),
	RunE: runAuthRefresh,
}

var (
	//Login options
	manualAuth bool
)

func init() {
	rootCmd.AddCommand(authCmd)
	authCmd.AddCommand(authLoginCmd)
	authCmd.AddCommand(authLogoutCmd)
	authCmd.AddCommand(authStatusCmd)
	authCmd.AddCommand(authShowCmd)
	authCmd.AddCommand(authRefreshCmd)

	//Login command options
	authLoginCmd.Flags().BoolVar(&manualAuth, "manual", false, "Manually enter the authorization code (for browser-less environments)")
}

// runAuthLogin performs login
func rejectAuthMachineOutput(command string) error {
	if globalProjectionModeRequested() || IsQuietMode() {
		return usageError(command + " does not support machine output; omit --json, --agent, --events, --explain, and --quiet")
	}
	return nil
}

func runAuthLogin(cmd *cobra.Command, args []string) error {
	if err := rejectAuthMachineOutput("auth login"); err != nil {
		return err
	}
	//Resolve Provider names (supports abbreviation)
	providerName := provider.ResolveProviderName(args[0])
	if !provider.IsValidProvider(providerName) {
		return usageError(fmt.Sprintf("Unsupported Provider: %s", args[0]))
	}

	switch providerName {
	case "google":
		return loginGoogle()
	case "microsoft":
		return loginMicrosoft()
	case "feishu":
		return loginFeishu()
	case "ticktick":
		return loginTickTick()
	case "dida":
		return loginDida()
	case "todoist":
		return loginTodoist()
	default:
		def, _ := provider.GetProviderDefinition(providerName)
		return commandError(fmt.Sprintf("%s has not yet implemented the login function", def.DisplayName), nil)
	}
}

// loginGoogle login Google
func loginGoogle() error {
	fmt.Println("🔐 Start Google Tasks OAuth2 authentication...")

	//Make sure the credentials directory exists
	if err := paths.EnsureCredentialsDir(); err != nil {
		fmt.Printf("❌ Failed to create credentials directory: %v\n", err)
		return commandError("Creation of credentials directory failed", err)
	}

	//Check the credentials file
	credentialsPath := paths.GetCredentialsPath("google")
	if _, err := os.Stat(credentialsPath); os.IsNotExist(err) {
		fmt.Printf("❌ Credential file does not exist: %s\n", credentialsPath)
		fmt.Println("\nPlease follow these steps:")
		fmt.Println("1. Visit Google Cloud Console: https://console.cloud.google.com/")
		fmt.Println("2. Create a project and enable Google Tasks API")
		fmt.Println("3. Configure the OAuth2 consent screen")
		fmt.Println("4. Create OAuth2 credentials (desktop app)")
		fmt.Printf("5. Download the certificate file and save it to: %s\n", credentialsPath)
		return commandError("Credential file does not exist", nil)
	}

	//Load credentials
	client, err := google.LoadCredentials(credentialsPath)
	if err != nil {
		fmt.Printf("❌ Failed to load credentials: %v\n", err)
		return commandError("Failed to load credentials", err)
	}

	//Set token file path
	tokenPath := paths.GetTokenPath("google")
	client.SetTokenFile(tokenPath)

	if manualAuth { //Generate authorization URL
		state := fmt.Sprintf("taskbridge-%d", time.Now().Unix())
		authURL := client.GetAuthURL(state)

		fmt.Println("\n📋 Please open the following link in your browser for authorization:")
		fmt.Println()
		fmt.Printf("   %s\n", authURL)
		fmt.Println()

		//Manually enter authorization code mode (supports directly pasting callback URL)
		fmt.Print("Please enter the authorization code (or paste the full callback URL):")
		reader := bufio.NewReader(os.Stdin)
		input, err := reader.ReadString('\n')
		if err != nil {
			fmt.Printf("❌ Failed to read authorization code: %v\n", err)
			return commandError("Failed to read authorization code", err)
		}
		code, err := extractGoogleAuthCode(input)
		if err != nil {
			fmt.Printf("❌ Authorization code format error: %v\n", err)
			fmt.Println("Please copy the complete value after `code=` in the browser callback address, or directly paste the complete callback URL.")
			return commandError("Authorization code format error", err)
		}

		//Exchange tokens
		token, err := client.Exchange(context.Background(), code)
		if err != nil {
			fmt.Printf("❌ Exchange token failed: %v\n", err)
			return commandError("Exchange token failed", err)
		}

		//save token
		if err := client.SaveToken(token); err != nil {
			fmt.Printf("❌ Failed to save token: %v\n", err)
			return commandError("Failed to save token", err)
		}

		fmt.Println("\n✅ Google Tasks authentication successful!")
		fmt.Printf("📁 Token saved to: %s\n", tokenPath)
	} else {
		//Automatic mode: Start the callback service locally to complete the authentication
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		token, err := client.StartAuthServer(ctx, 0)
		if err != nil {
			fmt.Printf("❌ Automatic authentication failed: %v\n", err)
			fmt.Println("You can switch to manual mode: taskbridge auth login google --manual")
			fmt.Println("If you use Google Desktop credentials and the redirect_uri is http://localhost, please make sure that port 80 of your local machine can listen.")
			return commandError("Automatic authentication failed", err)
		}

		fmt.Println("\n✅ Google Tasks automatic authentication successful!")
		fmt.Printf("📁 Token saved to: %s\n", tokenPath)
		if !token.Expiry.IsZero() {
			fmt.Printf("⏰ Expiration time: %s\n", token.Expiry.Format("2006-01-02 15:04:05"))
		}
	}
	return nil
}

func extractGoogleAuthCode(input string) (string, error) {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return "", fmt.Errorf("input is empty")
	}

	//Supports directly pasting the complete callback URL.
	if strings.HasPrefix(trimmed, "http://") || strings.HasPrefix(trimmed, "https://") {
		parsedURL, err := url.Parse(trimmed)
		if err != nil {
			return "", fmt.Errorf("unable to resolve URL: %w", err)
		}
		code := strings.TrimSpace(parsedURL.Query().Get("code"))
		if code == "" {
			return "", fmt.Errorf("code parameter not found in URL")
		}
		return code, nil
	}

	//Supports pasting query strings (e.g. code=xxx&scope=...).
	if strings.Contains(trimmed, "code=") {
		query := trimmed
		if idx := strings.Index(trimmed, "?"); idx >= 0 && idx < len(trimmed)-1 {
			query = trimmed[idx+1:]
		}
		values, err := url.ParseQuery(query)
		if err == nil {
			code := strings.TrimSpace(values.Get("code"))
			if code != "" {
				return code, nil
			}
		}
	}

	if strings.HasPrefix(trimmed, "taskbridge-") || looksLikeNumericState(trimmed) {
		return "", fmt.Errorf("it seems that the input is state, not authorization code")
	}

	return trimmed, nil
}

func looksLikeNumericState(v string) bool {
	if len(v) < 8 || len(v) > 16 {
		return false
	}
	for _, ch := range v {
		if ch < '0' || ch > '9' {
			return false
		}
	}
	return true
}

// runAuthLogout performs logout
func runAuthLogout(cmd *cobra.Command, args []string) error { //Resolve Provider names (supports abbreviation)
	providerName := provider.ResolveProviderName(args[0])

	//Check Provider yesnoefficient
	if !provider.IsValidProvider(providerName) {
		return usageError(fmt.Sprintf("Unsupported Provider: %s", args[0]))
	}

	tokenPath := paths.GetTokenPath(providerName)
	hasToken, err := tokenstore.Has(tokenPath, providerName)
	if err != nil {
		return commandError("Failed to read token", err)
	}

	receipt := AuthLogoutReceipt{
		Provider:    providerName,
		DisplayName: getAuthProviderMeta(providerName).DisplayName,
		TokenPath:   tokenPath,
		HadToken:    hasToken,
	}
	if !hasToken {
		projection := buildAuthLogoutProjection(receipt)
		return printProjection("text", projection, func() {
			fmt.Print(renderAuthLogout(projection))
		})
	}

	if err := tokenstore.Delete(tokenPath, providerName); err != nil {
		return commandError("Logout failed", err)
	}

	receipt.TokenDeleted = true
	projection := buildAuthLogoutProjection(receipt)
	return printProjection("text", projection, func() {
		fmt.Print(renderAuthLogout(projection))
	})
}

type AuthLogoutReceipt struct {
	Provider     string `json:"provider"`
	DisplayName  string `json:"display_name"`
	TokenPath    string `json:"token_path"`
	HadToken     bool   `json:"had_token"`
	TokenDeleted bool   `json:"token_deleted"`
}

func buildAuthLogoutProjection(receipt AuthLogoutReceipt) clioutput.Projection {
	projection := clioutput.New("auth.logout")
	projection.Facts["provider"] = receipt.Provider
	projection.Facts["had_token"] = receipt.HadToken
	projection.Facts["token_deleted"] = receipt.TokenDeleted
	projection.Facts["token_path"] = receipt.TokenPath
	projection.Data = receipt
	projection.Actions = []clioutput.Action{{Name: "login", Command: "taskbridge auth login " + receipt.Provider}}
	if receipt.TokenDeleted {
		projection.Summary = receipt.DisplayName + " is logged out."
	} else {
		projection.Summary = receipt.DisplayName + " was already logged out."
	}
	return projection
}

func renderAuthLogout(projection clioutput.Projection) string {
	receipt, _ := projection.Data.(AuthLogoutReceipt)
	return clioutput.RenderSummary(clioutput.Projection{
		SpecVersion: clioutput.SpecVersion,
		Command:     projection.Command,
		Status:      projection.Status,
		Summary:     projection.Summary,
		Facts: map[string]any{
			"Provider":      receipt.DisplayName,
			"Token file":    receipt.TokenPath,
			"Token existed": receipt.HadToken,
			"Token deleted": receipt.TokenDeleted,
		},
		Actions: projection.Actions,
	})
}

type AuthStatusRow struct {
	Provider      string `json:"provider"`
	DisplayName   string `json:"display_name"`
	Alias         string `json:"alias"`
	Status        string `json:"status"`
	Authenticated bool   `json:"authenticated"`
	Valid         string `json:"valid"`
	ExpiresAt     string `json:"expires_at"`
	NextAction    string `json:"next_action"`
}

func buildAuthStatusProjection() clioutput.Projection {
	rows := make([]AuthStatusRow, 0, len(getAuthProviderOrder()))
	authenticated := 0
	for _, p := range getAuthProviderOrder() {
		snapshot := getProviderAuthSnapshot(p)
		if snapshot.Authenticated {
			authenticated++
		}
		rows = append(rows, AuthStatusRow{
			Provider:      snapshot.Provider,
			DisplayName:   snapshot.DisplayName,
			Alias:         snapshot.ShortName,
			Status:        authDisplayStatus(snapshot),
			Authenticated: snapshot.Authenticated,
			Valid:         authValidity(snapshot.Valid),
			ExpiresAt:     authExpiry(snapshot.ExpiresAt),
			NextAction:    snapshot.NextAction,
		})
	}
	projection := clioutput.New("auth.status")
	projection.Summary = fmt.Sprintf("%d of %d providers are authenticated.", authenticated, len(rows))
	projection.Facts["providers"] = len(rows)
	projection.Facts["authenticated"] = authenticated
	for _, row := range rows {
		projection.Facts["provider."+row.Provider+".authenticated"] = row.Authenticated
		projection.Facts["provider."+row.Provider+".status"] = row.Status
	}
	projection.Data = map[string]any{"providers": rows}
	return projection
}

func renderAuthStatus(projection clioutput.Projection) string {
	data, _ := projection.Data.(map[string]any)
	rows, _ := data["providers"].([]AuthStatusRow)
	table := ui.NewTable("Provider", "Alias", "Status", "Expires")
	for _, row := range rows {
		table.AddRow(row.DisplayName, row.Alias, statusWithIcon(row.Status), row.ExpiresAt)
	}
	return "\n" + table.Render() + "\n"
}

// runAuthStatus executes state query
func runAuthStatus(cmd *cobra.Command, args []string) error {
	projection := buildAuthStatusProjection()
	return printProjection("text", projection, func() {
		fmt.Print(renderAuthStatus(projection))
	})
}

type AuthShowDetails struct {
	Provider      string `json:"provider"`
	DisplayName   string `json:"display_name"`
	Alias         string `json:"alias"`
	TokenPath     string `json:"token_path"`
	Status        string `json:"status"`
	Authenticated bool   `json:"authenticated"`
	Valid         string `json:"valid"`
	ExpiresAt     string `json:"expires_at"`
	NextAction    string `json:"next_action"`
}

func buildAuthShowProjection(providerName string) clioutput.Projection {
	snapshot := getProviderAuthSnapshot(providerName)
	details := AuthShowDetails{
		Provider:      snapshot.Provider,
		DisplayName:   snapshot.DisplayName,
		Alias:         snapshot.ShortName,
		TokenPath:     snapshot.TokenPath,
		Status:        authDisplayStatus(snapshot),
		Authenticated: snapshot.Authenticated,
		Valid:         authValidity(snapshot.Valid),
		ExpiresAt:     authExpiry(snapshot.ExpiresAt),
		NextAction:    snapshot.NextAction,
	}

	projection := clioutput.New("auth.show")
	projection.Summary = snapshot.DisplayName + " authentication details."
	projection.Facts["provider"] = providerName
	projection.Facts["authenticated"] = details.Authenticated
	projection.Facts["valid"] = details.Valid
	projection.Facts["expires_at"] = details.ExpiresAt
	projection.Data = details
	projection.Actions = []clioutput.Action{{Name: "next", Command: details.NextAction}}
	return projection
}

func renderAuthShow(projection clioutput.Projection) string {
	details, _ := projection.Data.(AuthShowDetails)
	return clioutput.RenderSummary(clioutput.Projection{
		SpecVersion: clioutput.SpecVersion,
		Command:     projection.Command,
		Status:      projection.Status,
		Summary:     projection.Summary,
		Facts: map[string]any{
			"Provider":       details.DisplayName,
			"Alias":          details.Alias,
			"Status":         statusWithIcon(details.Status),
			"Authenticated":  details.Authenticated,
			"Token is valid": details.Valid,
			"Expires":        details.ExpiresAt,
			"Token file":     details.TokenPath,
		},
		Actions: projection.Actions,
	})
}

// runAuthShow executes a single Provider details query
func runAuthShow(cmd *cobra.Command, args []string) error {
	providerName := provider.ResolveProviderName(args[0])
	if !provider.IsValidProvider(providerName) {
		return usageError(fmt.Sprintf("Unsupported Provider: %s", args[0]))
	}

	projection := buildAuthShowProjection(providerName)
	return printProjection("text", projection, func() {
		fmt.Print(renderAuthShow(projection))
	})
}

// runAuthRefresh performs token refresh
func runAuthRefresh(cmd *cobra.Command, args []string) error { //Resolve Provider names (supports abbreviation)
	if err := rejectAuthMachineOutput("auth refresh"); err != nil {
		return err
	}
	providerName := provider.ResolveProviderName(args[0])
	//Check Provider yesnoefficient
	if !provider.IsValidProvider(providerName) {
		return usageError(fmt.Sprintf("Unsupported Provider: %s", args[0]))
	}

	switch providerName {
	case "google":
		return refreshGoogleToken()
	case "microsoft":
		return refreshMicrosoftToken()
	case "feishu":
		return refreshFeishuToken()
	case "ticktick":
		return refreshTickTickToken()
	case "dida":
		return refreshDidaToken()
	case "todoist":
		return refreshTodoistToken()
	default:
		def, _ := provider.GetProviderDefinition(providerName)
		return commandError(fmt.Sprintf("%s has not yet implemented the token refresh function", def.DisplayName), nil)
	}
}

// refreshGoogleToken refresh Google token
func refreshGoogleToken() error {
	client, err := google.NewOAuth2ClientFromHome()
	if err != nil {
		fmt.Printf("❌ Failed to load Google OAuth2 client: %v\n", err)
		return commandError("Failed to load Google OAuth2 client", err)
	}

	token, err := client.RefreshToken(context.Background())
	if err != nil {
		fmt.Printf("❌ Failed to refresh token: %v\n", err)
		return commandError("Refresh token failed", err)
	}

	if err := client.SaveToken(token); err != nil {
		fmt.Printf("❌ Failed to save token: %v\n", err)
		return commandError("Failed to save token", err)
	}

	fmt.Println("✅ Google token has been refreshed")
	return nil
}

// loginMicrosoft login Microsoft
func loginMicrosoft() error {
	fmt.Println("🔐 Start Microsoft To Do OAuth2 Authentication...")

	//Make sure the credentials directory exists
	if err := paths.EnsureCredentialsDir(); err != nil {
		fmt.Printf("❌ Failed to create credentials directory: %v\n", err)
		return commandError("Creation of credentials directory failed", err)
	}

	//Check the credentials file
	credentialsPath := paths.GetCredentialsPath("microsoft")
	if _, err := os.Stat(credentialsPath); os.IsNotExist(err) {
		fmt.Printf("❌ Credential file does not exist: %s\n", credentialsPath)
		fmt.Println("\nPlease follow these steps:")
		fmt.Println("1. Visit Azure Portal: https://portal.azure.com/")
		fmt.Println("2. Register the application (Azure Active Directory)")
		fmt.Println("3. Configure redirect URI: http://localhost:8080/callback")
		fmt.Println("4. Add API permissions: Tasks.ReadWrite, User.Read")
		fmt.Println("5. Create client key")
		fmt.Printf("6. Create the credentials file and save it to: %s\n", credentialsPath)
		fmt.Println("\nCertificate file format:")
		fmt.Println(`{
	 "client_id": "your app id",
	 "client_secret": "your client key",
	 "tenant_id": "common",
	 "redirect_url": "http://localhost:8080/callback"
}`)
		return commandError("Credential file does not exist", nil)
	}

	//Load credentials
	oauthClient, err := microsoft.LoadCredentials(credentialsPath)
	if err != nil {
		fmt.Printf("❌ Failed to load credentials: %v\n", err)
		return commandError("Failed to load credentials", err)
	}

	//Set token file path
	tokenPath := paths.GetTokenPath("microsoft")
	oauthClient.SetTokenFile(tokenPath)

	//Start authentication server
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	token, err := oauthClient.StartAuthServer(ctx, 8080)
	if err != nil {
		fmt.Printf("❌ Authentication failed: %v\n", err)
		return commandError("Authentication failed", err)
	}

	fmt.Println("\n✅ Microsoft To Do certification successful!")
	fmt.Printf("📁 Token saved to: %s\n", tokenPath)
	fmt.Printf("🔑 Token type: %s\n", token.TokenType)
	return nil
}

// refreshMicrosoftToken refresh Microsoft token
func refreshMicrosoftToken() error {
	credentialsPath := paths.GetCredentialsPath("microsoft")
	tokenPath := paths.GetTokenPath("microsoft")

	oauthClient, err := microsoft.LoadCredentials(credentialsPath)
	if err != nil {
		fmt.Printf("❌ Failed to load Microsoft OAuth2 client: %v\n", err)
		return commandError("Failed to load Microsoft OAuth2 client", err)
	}

	oauthClient.SetTokenFile(tokenPath)

	//Load existing token
	if err := oauthClient.LoadToken(); err != nil {
		fmt.Printf("❌ Failed to load token: %v\n", err)
		return commandError("Failed to load token", err)
	}

	//refresh token
	token, err := oauthClient.RefreshToken(context.Background())
	if err != nil {
		fmt.Printf("❌ Failed to refresh token: %v\n", err)
		return commandError("Refresh token failed", err)
	}

	if err := oauthClient.SaveToken(); err != nil {
		fmt.Printf("❌ Failed to save token: %v\n", err)
		return commandError("Failed to save token", err)
	}

	fmt.Println("✅ Microsoft token has been refreshed")
	fmt.Printf("🔑 New expiration time: %s\n", token.Expiry.Format("2006-01-02 15:04:05"))
	return nil
}

// loginFeishu login Feishu
func loginFeishu() error {
	fmt.Println("🔐 Start Feishu Todo OAuth2 authentication...")

	if err := paths.EnsureCredentialsDir(); err != nil {
		fmt.Printf("❌ Failed to create credentials directory: %v\n", err)
		return commandError("Creation of credentials directory failed", err)
	}

	credentialsPath := paths.GetCredentialsPath("feishu")
	if _, err := os.Stat(credentialsPath); os.IsNotExist(err) {
		fmt.Printf("❌ Credential file does not exist: %s\n", credentialsPath)
		fmt.Println("\nPlease follow these steps:")
		fmt.Println("1. Visit Feishu open platform: https://open.feishu.cn/")
		fmt.Println("2. Create a self-built application and enable Todo related permissions")
		fmt.Println("3. Configure the redirect URL (the port must be consistent with the local callback listening, such as http://127.0.0.1:3456/callback)")
		fmt.Printf("4. Create the credentials file and save it to: %s\n", credentialsPath)
		fmt.Println("\nCertificate file format:")
		fmt.Println(`{
  "app_id": "cli_xxx",
  "app_secret": "xxxx",
  "redirect_url": "http://127.0.0.1:3456/callback",
  "scopes": ["task:tasklist:read","task:tasklist:write","task:task:read","task:task:write"]
}`)
		return commandError("Credential file does not exist", nil)
	}

	oauthClient, err := feishu.LoadCredentials(credentialsPath)
	if err != nil {
		fmt.Printf("❌ Failed to load credentials: %v\n", err)
		return commandError("Failed to load credentials", err)
	}

	tokenPath := paths.GetTokenPath("feishu")
	oauthClient.SetTokenFile(tokenPath)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	//The port is determined by the redirect_url in the credentials (8080 is not mandatory)
	token, err := oauthClient.StartAuthServer(ctx, 0)
	if err != nil {
		fmt.Printf("❌ Authentication failed: %v\n", err)
		return commandError("Authentication failed", err)
	}

	fmt.Println("\n✅ Feishu Todo certification successful!")
	fmt.Printf("📁 Token saved to: %s\n", tokenPath)
	fmt.Printf("🔑 Token type: %s\n", token.TokenType)
	return nil
}

// refreshFeishuToken refresh Feishu token
func refreshFeishuToken() error {
	credentialsPath := paths.GetCredentialsPath("feishu")
	tokenPath := paths.GetTokenPath("feishu")

	oauthClient, err := feishu.LoadCredentials(credentialsPath)
	if err != nil {
		fmt.Printf("❌ Failed to load Feishu OAuth2 client: %v\n", err)
		return commandError("Failed to load Feishu OAuth2 client", err)
	}

	oauthClient.SetTokenFile(tokenPath)
	if err := oauthClient.LoadToken(); err != nil {
		fmt.Printf("❌ Failed to load token: %v\n", err)
		return commandError("Failed to load token", err)
	}

	token, err := oauthClient.RefreshToken(context.Background())
	if err != nil {
		fmt.Printf("❌ Failed to refresh token: %v\n", err)
		return commandError("Refresh token failed", err)
	}

	if err := oauthClient.SaveToken(); err != nil {
		fmt.Printf("❌ Failed to save token: %v\n", err)
		return commandError("Failed to save token", err)
	}

	fmt.Println("✅ Feishu token has been refreshed")
	fmt.Printf("🔑 Expires in: %d seconds\n", token.ExpiresIn)
	return nil
}

// loginTickTick login TickTick (API Token)
func loginTickTick() error {
	return loginTickStyleProvider("ticktick")
}

func loginDida() error {
	return loginTickStyleProvider("dida")
}

func loginTickStyleProvider(providerName string) error {
	displayName := "TickTick"
	tokenHint := "tp_"
	if providerName == "dida" {
		displayName = "Dida365"
		tokenHint = "dp_"
	}

	fmt.Printf("🔐 Start %s API Token authentication...\n", displayName)

	if err := paths.EnsureCredentialsDir(); err != nil {
		fmt.Printf("❌ Failed to create credentials directory: %v\n", err)
		return commandError("Creation of credentials directory failed", err)
	}

	tokenPath := paths.GetTokenPath(providerName)
	fmt.Printf("\nPlease follow the steps below to obtain %s API Token:\n", displayName)
	if providerName == "dida" {
		fmt.Println("1. Open dida365.com and log in to the developer platform or OpenAPI management page")
	} else {
		fmt.Println("1. Open the TickTick Developer Platform and log in")
	}
	fmt.Println("2. Create or view personal API Token")
	fmt.Printf("3. Copy token (usually starts with `%s`)\n", tokenHint)
	fmt.Println()

	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("Please enter %s API Token:", displayName)
	apiToken, err := reader.ReadString('\n')
	if err != nil {
		fmt.Printf("❌ Failed to read API Token: %v\n", err)
		return commandError("Failed to read API Token", err)
	}
	apiToken = strings.TrimSpace(apiToken)
	if apiToken == "" {
		fmt.Println("❌ API Token cannot be empty")
		return commandError("API Token cannot be empty", nil)
	}

	p, err := ticktick.NewProvider(ticktick.Config{
		ProviderName: providerName,
		Token:        apiToken,
		TokenFile:    tokenPath,
	})
	if err != nil {
		fmt.Printf("❌ Failed to initialize %s Provider: %v\n", displayName, err)
		return commandError("Failed to initialize Provider", err)
	}
	if err := p.Authenticate(context.Background(), map[string]interface{}{
		"token":    apiToken,
		"provider": providerName,
	}); err != nil {
		fmt.Printf("❌ %s Authentication failed: %v\n", displayName, err)
		return commandError("Authentication failed", err)
	}

	fmt.Printf("\n✅ %s authentication successful!\n", displayName)
	fmt.Printf("📁 Token saved to: %s\n", tokenPath)
	return nil
}

// refreshTickTickToken refresh TickTick token (static token)
func refreshTickTickToken() error {
	return refreshTickStyleProvider("ticktick")
}

func refreshDidaToken() error {
	return refreshTickStyleProvider("dida")
}

func refreshTickStyleProvider(providerName string) error {
	displayName := "TickTick"
	if providerName == "dida" {
		displayName = "Dida365"
	}
	tokenPath := paths.GetTokenPath(providerName)
	hasToken, err := tokenstore.Has(tokenPath, providerName)
	if err != nil {
		fmt.Printf("❌ Failed to read %s token: %v\n", displayName, err)
		return commandError("Failed to read token", err)
	}
	if !hasToken {
		fmt.Printf("❌ The %s credentials do not exist, please execute: taskbridge auth login %s\n", displayName, providerName)
		return commandError("The certificate does not exist, please log in first", nil)
	}

	p, err := ticktick.NewProvider(ticktick.Config{
		ProviderName: providerName,
		TokenFile:    tokenPath,
	})
	if err != nil {
		fmt.Printf("❌ Failed to initialize %s Provider: %v\n", displayName, err)
		return commandError("Failed to initialize Provider", err)
	}
	if err := p.RefreshToken(context.Background()); err != nil {
		fmt.Printf("❌ Failed to refresh %s token: %v\n", displayName, err)
		return commandError("Refresh token failed", err)
	}

	fmt.Printf("✅ %s token verification passed (static token does not need to be refreshed)\n", displayName)
	return nil
}

// loginTodoist login Todoist (API Token)
func loginTodoist() error {
	fmt.Println("🔐 Start Todoist API Token authentication...")

	if err := paths.EnsureCredentialsDir(); err != nil {
		fmt.Printf("❌ Failed to create credentials directory: %v\n", err)
		return commandError("Creation of credentials directory failed", err)
	}

	tokenPath := paths.GetTokenPath("todoist")
	fmt.Println("\nPlease follow the steps below to obtain Todoist API Token:")
	fmt.Println("1. Visit https://todoist.com/app/settings/integrations/developer")
	fmt.Println("2. Copy API Token")
	fmt.Println()

	fmt.Print("Please enter API Token:")
	reader := bufio.NewReader(os.Stdin)
	token, err := reader.ReadString('\n')
	if err != nil {
		fmt.Printf("❌ Failed to read API Token: %v\n", err)
		return commandError("Failed to read API Token", err)
	}
	token = strings.TrimSpace(token)
	if token == "" {
		fmt.Println("❌ API Token cannot be empty")
		return commandError("API Token cannot be empty", nil)
	}

	p, err := todoist.NewProvider(todoist.Config{
		APIToken:  token,
		TokenFile: tokenPath,
	})
	if err != nil {
		fmt.Printf("❌ Failed to initialize Todoist Provider: %v\n", err)
		return commandError("Failed to initialize Provider", err)
	}
	if err := p.Authenticate(context.Background(), map[string]interface{}{"api_token": token}); err != nil {
		fmt.Printf("❌ Todoist Authentication failed: %v\n", err)
		return commandError("Authentication failed", err)
	}

	fmt.Println("\n✅ Todoist certification successful!")
	fmt.Printf("📁 Token saved to: %s\n", tokenPath)
	return nil
}

// refreshTodoistToken refresh Todoist token (static API Token, no Need to refresh)
func refreshTodoistToken() error {
	fmt.Println("ℹ️ Todoist uses static API Token, no need to refresh.")
	fmt.Println("If the token is invalid, please re-execute: taskbridge auth login todoist")
	return nil
}

type AuthSnapshot struct {
	Provider      string
	DisplayName   string
	ShortName     string
	TokenPath     string
	Authenticated bool
	Valid         *bool
	StatusText    string
	ExpiresAt     string
	NextAction    string
}

type providerMeta struct {
	Name        string
	DisplayName string
	ShortName   string
}

func getAuthProviderOrder() []string {
	return provider.GetAllProviderNames()
}

func getAuthProviderMeta(name string) providerMeta {
	def, ok := provider.GetProviderDefinition(name)
	if !ok {
		return providerMeta{
			Name:        name,
			DisplayName: name,
			ShortName:   name,
		}
	}
	return providerMeta{
		Name:        def.Name,
		DisplayName: def.DisplayName,
		ShortName:   def.ShortName,
	}
}

func boolPtr(v bool) *bool {
	return &v
}

func authValidity(valid *bool) string {
	if valid == nil {
		return "unknown"
	}
	if *valid {
		return "yes"
	}
	return "no"
}

func authDisplayStatus(snapshot AuthSnapshot) string {
	if snapshot.Authenticated {
		if snapshot.Valid != nil && !*snapshot.Valid {
			return "Expired"
		}
		return "Connected"
	}
	if snapshot.StatusText == "⚠️ Token error" {
		return "Token error"
	}
	if isProviderEnabled(snapshot.Provider) {
		return "Not authenticated"
	}
	return "Not configured"
}

func authExpiry(value string) string {
	if value == "efficient" {
		return "valid"
	}
	return value
}

func isProviderEnabled(providerName string) bool {
	switch providerName {
	case "google":
		return cfg.Providers.Google.Enabled
	case "microsoft":
		return cfg.Providers.Microsoft.Enabled
	case "feishu":
		return cfg.Providers.Feishu.Enabled
	case "ticktick":
		return cfg.Providers.TickTick.Enabled
	case "dida":
		return cfg.Providers.Dida.Enabled
	case "todoist":
		return cfg.Providers.Todoist.Enabled
	default:
		return false
	}
}

func getProviderAuthSnapshot(providerName string) AuthSnapshot {
	meta := getAuthProviderMeta(providerName)
	snapshot := AuthSnapshot{
		Provider:      meta.Name,
		DisplayName:   meta.DisplayName,
		ShortName:     meta.ShortName,
		TokenPath:     paths.GetTokenPath(meta.Name),
		Authenticated: false,
		Valid:         nil,
		StatusText:    "❌ Not configured",
		ExpiresAt:     "-",
		NextAction:    fmt.Sprintf("taskbridge auth login %s", meta.Name),
	}

	hasToken, err := tokenstore.Has(snapshot.TokenPath, meta.Name)
	if err != nil {
		snapshot.StatusText = "⚠️ Token error"
		snapshot.ExpiresAt = "Read failed"
		snapshot.NextAction = fmt.Sprintf("Check token file: %s", snapshot.TokenPath)
		return snapshot
	}
	if !hasToken {
		if isProviderEnabled(meta.Name) {
			snapshot.StatusText = "❌ Not authenticated"
		}
		return snapshot
	}

	snapshot.Authenticated = true
	snapshot.StatusText = "✅ Connected"
	snapshot.NextAction = fmt.Sprintf("taskbridge auth logout %s", meta.Name)

	switch meta.Name {
	case "google":
		client, err := google.NewOAuth2ClientFromHome()
		if err != nil {
			//When the Google client fails to load, it is conservatively considered connected but the efficiency cannot be determined.
			snapshot.Valid = nil
			snapshot.ExpiresAt = "efficient"
			return snapshot
		}
		info := client.GetTokenInfo()
		if info == nil {
			snapshot.Valid = nil
			snapshot.ExpiresAt = "unknown"
			return snapshot
		}
		if info.Valid {
			snapshot.Valid = boolPtr(true)
			snapshot.ExpiresAt = info.Expiry.Format("2006-01-02")
			return snapshot
		}
		snapshot.Valid = boolPtr(false)
		snapshot.StatusText = "⚠️ Expired"
		snapshot.ExpiresAt = "Need to refresh"
		snapshot.NextAction = "taskbridge auth refresh google"
		return snapshot
	case "microsoft":
		credentialsPath := paths.GetCredentialsPath("microsoft")
		oauthClient, err := microsoft.LoadCredentials(credentialsPath)
		if err != nil {
			snapshot.Valid = nil
			snapshot.ExpiresAt = "efficient"
			return snapshot
		}
		oauthClient.SetTokenFile(snapshot.TokenPath)
		if err := oauthClient.LoadToken(); err != nil {
			snapshot.Valid = nil
			snapshot.ExpiresAt = "efficient"
			return snapshot
		}
		info := oauthClient.GetTokenInfo()
		if expiry, ok := info["expiry"].(time.Time); ok && !expiry.IsZero() {
			snapshot.ExpiresAt = expiry.Format("2006-01-02")
			now := time.Now()
			if expiry.After(now) {
				snapshot.Valid = boolPtr(true)
			} else {
				snapshot.Valid = boolPtr(false)
				snapshot.StatusText = "⚠️ Expired"
				snapshot.NextAction = "taskbridge auth refresh microsoft"
			}
			return snapshot
		}
		snapshot.Valid = nil
		snapshot.ExpiresAt = "efficient"
		return snapshot
	default:
		snapshot.Valid = nil
		snapshot.ExpiresAt = "efficient"
		return snapshot
	}
}
