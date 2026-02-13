package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/openoms-org/openoms/apps/api-server/internal/database"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
)

// AIService provides AI-powered product categorization and description generation.
type AIService struct {
	apiKey     string
	model      string
	httpClient *http.Client

	productRepo repository.ProductRepo
	tenantRepo  repository.TenantRepo
	pool        *pgxpool.Pool
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	Temperature float64       `json:"temperature"`
}

type chatChoice struct {
	Message chatMessage `json:"message"`
}

type chatResponse struct {
	Choices []chatChoice `json:"choices"`
}

// NewAIService creates a new AI service. If apiKey is empty the service will
// return ErrAINotConfigured on every call.
func NewAIService(apiKey, model string, productRepo repository.ProductRepo, tenantRepo repository.TenantRepo, pool *pgxpool.Pool) *AIService {
	return &AIService{
		apiKey: apiKey,
		model:  model,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
		productRepo: productRepo,
		tenantRepo:  tenantRepo,
		pool:        pool,
	}
}

// IsConfigured returns true when the OpenAI API key is set.
func (s *AIService) IsConfigured() bool {
	return s.apiKey != ""
}

func (s *AIService) callOpenAI(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	if !s.IsConfigured() {
		return "", fmt.Errorf("AI nie jest skonfigurowane")
	}

	reqBody := chatRequest{
		Model: s.model,
		Messages: []chatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		Temperature: 0.3,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.openai.com/v1/chat/completions", bytes.NewReader(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.apiKey)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("openai request: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("openai error (status %d): %s", resp.StatusCode, string(respBytes))
	}

	var chatResp chatResponse
	if err := json.Unmarshal(respBytes, &chatResp); err != nil {
		return "", fmt.Errorf("unmarshal response: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("empty response from OpenAI")
	}

	return strings.TrimSpace(chatResp.Choices[0].Message.Content), nil
}

// SuggestCategories returns a list of suggested category keys for a product.
func (s *AIService) SuggestCategories(ctx context.Context, tenantID uuid.UUID, productName, productDescription string) ([]string, error) {
	// Fetch existing tenant categories to constrain the suggestion
	var categoriesJSON []byte
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		settings, err := s.tenantRepo.GetSettings(ctx, tx, tenantID)
		if err != nil {
			return err
		}
		categoriesJSON = settings
		return nil
	})
	if err != nil {
		return nil, err
	}

	var existingCats []string
	if categoriesJSON != nil {
		var allSettings map[string]json.RawMessage
		if err := json.Unmarshal(categoriesJSON, &allSettings); err == nil {
			if raw, ok := allSettings["product_categories"]; ok {
				var cfg struct {
					Categories []struct {
						Key   string `json:"key"`
						Label string `json:"label"`
					} `json:"categories"`
				}
				if err := json.Unmarshal(raw, &cfg); err == nil {
					for _, c := range cfg.Categories {
						existingCats = append(existingCats, c.Key+":"+c.Label)
					}
				}
			}
		}
	}

	catList := "brak zdefiniowanych kategorii"
	if len(existingCats) > 0 {
		catList = strings.Join(existingCats, ", ")
	}

	systemPrompt := "Jestes asystentem kategoryzacji produktow e-commerce. Zwracasz TYLKO surowy JSON, bez markdown."
	userPrompt := fmt.Sprintf(
		`Produkt: "%s"
Opis: "%s"
Dostepne kategorie (klucz:etykieta): %s

Zasugeruj 1-3 najbardziej pasujace kategorie. Jesli zadna nie pasuje, zasugeruj now.
Zwroc JSON: {"categories": ["klucz1", "klucz2"]}`,
		productName, productDescription, catList,
	)

	result, err := s.callOpenAI(ctx, systemPrompt, userPrompt)
	if err != nil {
		return nil, err
	}

	var parsed struct {
		Categories []string `json:"categories"`
	}
	if err := json.Unmarshal([]byte(result), &parsed); err != nil {
		return []string{}, nil
	}
	return parsed.Categories, nil
}

// SuggestTags returns a list of suggested tags for a product.
func (s *AIService) SuggestTags(ctx context.Context, productName, productDescription string) ([]string, error) {
	systemPrompt := "Jestes asystentem tagowania produktow e-commerce. Zwracasz TYLKO surowy JSON, bez markdown."
	userPrompt := fmt.Sprintf(
		`Produkt: "%s"
Opis: "%s"

Zasugeruj 3-5 trafionych tagow dla tego produktu. Tagi powinny byc krotkie (1-2 slowa), po polsku, male litery.
Zwroc JSON: {"tags": ["tag1", "tag2", "tag3"]}`,
		productName, productDescription,
	)

	result, err := s.callOpenAI(ctx, systemPrompt, userPrompt)
	if err != nil {
		return nil, err
	}

	var parsed struct {
		Tags []string `json:"tags"`
	}
	if err := json.Unmarshal([]byte(result), &parsed); err != nil {
		return []string{}, nil
	}
	return parsed.Tags, nil
}

// GenerateDescription generates a long product description.
func (s *AIService) GenerateDescription(ctx context.Context, productName, shortDescription string) (string, error) {
	systemPrompt := "Jestes copywriterem e-commerce. Piszesz opisy produktow po polsku. Zwracasz TYLKO surowy JSON, bez markdown."
	userPrompt := fmt.Sprintf(
		`Produkt: "%s"
Krotki opis: "%s"

Napisz atrakcyjny, szczegolowy opis produktu (3-5 zdan). Opis powinien byc po polsku, zachecajacy do zakupu.
Zwroc JSON: {"description": "tresc opisu"}`,
		productName, shortDescription,
	)

	result, err := s.callOpenAI(ctx, systemPrompt, userPrompt)
	if err != nil {
		return "", err
	}

	var parsed struct {
		Description string `json:"description"`
	}
	if err := json.Unmarshal([]byte(result), &parsed); err != nil {
		return result, nil
	}
	return parsed.Description, nil
}

// DescribeOptions holds optional parameters for enhanced description generation.
type DescribeOptions struct {
	Style       string `json:"style"`       // "professional" | "promotional" | "casual" | "seo"
	Language    string `json:"language"`    // "pl" | "en" | "de"
	Length      string `json:"length"`      // "short" | "medium" | "long"
	Marketplace string `json:"marketplace"` // "" | "allegro" | "amazon" | "ebay"
}

// AISuggestion holds all AI-generated suggestions for a product.
type AISuggestion struct {
	ProductID        uuid.UUID `json:"product_id"`
	Categories       []string  `json:"categories"`
	Tags             []string  `json:"tags"`
	Description      string    `json:"description,omitempty"`
	ShortDescription string    `json:"short_description,omitempty"`
	LongDescription  string    `json:"long_description,omitempty"`
}

// AITextResult holds a single text result from AI operations.
type AITextResult struct {
	Description string `json:"description"`
}

// Categorize fetches a product and returns AI suggestions without modifying the product.
func (s *AIService) Categorize(ctx context.Context, tenantID, productID uuid.UUID) (*AISuggestion, error) {
	var name, shortDesc string

	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		p, err := s.productRepo.FindByID(ctx, tx, productID)
		if err != nil {
			return err
		}
		if p == nil {
			return ErrProductNotFound
		}
		name = p.Name
		shortDesc = p.DescriptionShort
		return nil
	})
	if err != nil {
		return nil, err
	}

	categories, catErr := s.SuggestCategories(ctx, tenantID, name, shortDesc)
	tags, tagErr := s.SuggestTags(ctx, name, shortDesc)

	if catErr != nil && tagErr != nil {
		return nil, catErr
	}
	if catErr != nil {
		categories = []string{}
	}
	if tagErr != nil {
		tags = []string{}
	}

	return &AISuggestion{
		ProductID:  productID,
		Categories: categories,
		Tags:       tags,
	}, nil
}

// Describe fetches a product and returns an AI-generated description.
func (s *AIService) Describe(ctx context.Context, tenantID, productID uuid.UUID, opts *DescribeOptions) (*AISuggestion, error) {
	var name, shortDesc string

	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		p, err := s.productRepo.FindByID(ctx, tx, productID)
		if err != nil {
			return err
		}
		if p == nil {
			return ErrProductNotFound
		}
		name = p.Name
		shortDesc = p.DescriptionShort
		return nil
	})
	if err != nil {
		return nil, err
	}

	if opts == nil {
		opts = &DescribeOptions{}
	}
	if opts.Style == "" {
		opts.Style = "professional"
	}
	if opts.Language == "" {
		opts.Language = "pl"
	}
	if opts.Length == "" {
		opts.Length = "medium"
	}

	shortDescResult, longDescResult, err := s.GenerateEnhancedDescription(ctx, name, shortDesc, *opts)
	if err != nil {
		return nil, err
	}

	return &AISuggestion{
		ProductID:        productID,
		ShortDescription: shortDescResult,
		LongDescription:  longDescResult,
		Description:      longDescResult, // backward compatibility
	}, nil
}

// GenerateEnhancedDescription generates both short and long product descriptions with options.
func (s *AIService) GenerateEnhancedDescription(ctx context.Context, productName, shortDescription string, opts DescribeOptions) (string, string, error) {
	langMap := map[string]string{
		"pl": "po polsku",
		"en": "in English",
		"de": "auf Deutsch",
	}
	lang := langMap[opts.Language]
	if lang == "" {
		lang = "po polsku"
	}

	styleMap := map[string]string{
		"professional": "profesjonalny, rzeczowy ton",
		"promotional":  "promocyjny, zachecajacy do zakupu ton",
		"casual":       "swobodny, przyjazny ton",
		"seo":          "zoptymalizowany pod SEO, z uzyciem slow kluczowych",
	}
	style := styleMap[opts.Style]
	if style == "" {
		style = styleMap["professional"]
	}

	lengthMap := map[string]string{
		"short":  "krotki (1-2 zdania)",
		"medium": "sredni (3-5 zdan)",
		"long":   "dlugi (6-10 zdan, szczegolowy)",
	}
	length := lengthMap[opts.Length]
	if length == "" {
		length = lengthMap["medium"]
	}

	marketplaceHint := ""
	if opts.Marketplace != "" {
		marketplaceMap := map[string]string{
			"allegro": "Sformatuj opis pod Allegro - uzyj punktow, podkresl najwazniejsze cechy, dodaj sekcje 'Specyfikacja' i 'W zestawie'.",
			"amazon":  "Sformatuj opis pod Amazon - uzyj bullet pointow (max 5), zachowaj zwiezlosc, podkresl USP produktu.",
			"ebay":    "Sformatuj opis pod eBay - uzyj czytelnego formatowania HTML, dodaj sekcje opisowa i specyfikacje.",
		}
		marketplaceHint = marketplaceMap[opts.Marketplace]
		if marketplaceHint == "" {
			marketplaceHint = ""
		}
	}

	systemPrompt := fmt.Sprintf("Jestes copywriterem e-commerce. Piszesz opisy produktow %s. Styl: %s. Zwracasz TYLKO surowy JSON, bez markdown.", lang, style)

	marketplaceInstruction := ""
	if marketplaceHint != "" {
		marketplaceInstruction = fmt.Sprintf("\n%s", marketplaceHint)
	}

	userPrompt := fmt.Sprintf(
		`Produkt: "%s"
Krotki opis: "%s"

Napisz:
1. Krotki opis produktu (1-2 zdania, do 200 znakow)
2. Dlugi opis produktu (%s)

Jezyk: %s
%s
Zwroc JSON: {"short_description": "...", "long_description": "..."}`,
		productName, shortDescription, length, lang, marketplaceInstruction,
	)

	result, err := s.callOpenAI(ctx, systemPrompt, userPrompt)
	if err != nil {
		return "", "", err
	}

	var parsed struct {
		ShortDescription string `json:"short_description"`
		LongDescription  string `json:"long_description"`
	}
	if err := json.Unmarshal([]byte(result), &parsed); err != nil {
		// Fallback: try old format
		var oldParsed struct {
			Description string `json:"description"`
		}
		if err2 := json.Unmarshal([]byte(result), &oldParsed); err2 == nil {
			return "", oldParsed.Description, nil
		}
		return "", result, nil
	}
	return parsed.ShortDescription, parsed.LongDescription, nil
}

// ImproveDescription takes an existing description and improves it.
func (s *AIService) ImproveDescription(ctx context.Context, description, style, language string) (string, error) {
	if style == "" {
		style = "professional"
	}
	if language == "" {
		language = "pl"
	}

	langMap := map[string]string{
		"pl": "po polsku",
		"en": "in English",
		"de": "auf Deutsch",
	}
	lang := langMap[language]
	if lang == "" {
		lang = "po polsku"
	}

	styleMap := map[string]string{
		"professional": "profesjonalny, rzeczowy",
		"promotional":  "promocyjny, zachecajacy",
		"casual":       "swobodny, przyjazny",
		"seo":          "zoptymalizowany pod SEO",
	}
	styleTxt := styleMap[style]
	if styleTxt == "" {
		styleTxt = styleMap["professional"]
	}

	systemPrompt := fmt.Sprintf("Jestes copywriterem e-commerce. Poprawiasz opisy produktow. Jezyk: %s. Styl: %s. Zwracasz TYLKO surowy JSON, bez markdown.", lang, styleTxt)
	userPrompt := fmt.Sprintf(
		`Obecny opis produktu:
"%s"

Popraw ten opis - ulepsz styl, popraw bledy, uczyni go bardziej atrakcyjnym dla klientow. Zachowaj kluczowe informacje.
Zwroc JSON: {"description": "poprawiony opis"}`,
		description,
	)

	result, err := s.callOpenAI(ctx, systemPrompt, userPrompt)
	if err != nil {
		return "", err
	}

	var parsed struct {
		Description string `json:"description"`
	}
	if err := json.Unmarshal([]byte(result), &parsed); err != nil {
		return result, nil
	}
	return parsed.Description, nil
}

// TranslateDescription translates a product description to the target language.
func (s *AIService) TranslateDescription(ctx context.Context, description, targetLanguage string) (string, error) {
	langMap := map[string]string{
		"pl": "polski",
		"en": "angielski",
		"de": "niemiecki",
	}
	lang := langMap[targetLanguage]
	if lang == "" {
		lang = targetLanguage
	}

	systemPrompt := "Jestes profesjonalnym tlumaczem opisow produktow e-commerce. Zwracasz TYLKO surowy JSON, bez markdown."
	userPrompt := fmt.Sprintf(
		`Przetlumacz ponizszy opis produktu na jezyk %s. Zachowaj formatowanie i styl.

Opis do przetlumaczenia:
"%s"

Zwroc JSON: {"description": "przetlumaczony opis"}`,
		lang, description,
	)

	result, err := s.callOpenAI(ctx, systemPrompt, userPrompt)
	if err != nil {
		return "", err
	}

	var parsed struct {
		Description string `json:"description"`
	}
	if err := json.Unmarshal([]byte(result), &parsed); err != nil {
		return result, nil
	}
	return parsed.Description, nil
}
