package model

import (
	"errors"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"
	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

const MaxCategoryDepth = 5

type ProductCategory struct {
	ID        uuid.UUID         `json:"id"`
	TenantID  uuid.UUID         `json:"tenant_id"`
	ParentID  *uuid.UUID        `json:"parent_id,omitempty"`
	Name      string            `json:"name"`
	Slug      string            `json:"slug"`
	Color     string            `json:"color"`
	Icon      *string           `json:"icon,omitempty"`
	Position  int               `json:"position"`
	Depth     int               `json:"depth"`
	Children  []ProductCategory `json:"children,omitempty"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
}

type CreateCategoryRequest struct {
	Name     string     `json:"name"`
	ParentID *uuid.UUID `json:"parent_id,omitempty"`
	Color    string     `json:"color,omitempty"`
	Icon     *string    `json:"icon,omitempty"`
}

func (r *CreateCategoryRequest) Validate() error {
	if strings.TrimSpace(r.Name) == "" {
		return errors.New("name is required")
	}
	if err := validateMaxLength("name", r.Name, 200); err != nil {
		return err
	}
	if r.Color == "" {
		r.Color = "#6b7280"
	}
	return nil
}

type UpdateCategoryRequest struct {
	Name     *string    `json:"name,omitempty"`
	ParentID *uuid.UUID `json:"parent_id,omitempty"`
	Color    *string    `json:"color,omitempty"`
	Icon     *string    `json:"icon,omitempty"`
	Position *int       `json:"position,omitempty"`
}

func (r *UpdateCategoryRequest) Validate() error {
	if r.Name == nil && r.ParentID == nil && r.Color == nil && r.Icon == nil && r.Position == nil {
		return errors.New("at least one field must be provided")
	}
	if r.Name != nil {
		if strings.TrimSpace(*r.Name) == "" {
			return errors.New("name must not be empty")
		}
		if err := validateMaxLength("name", *r.Name, 200); err != nil {
			return err
		}
	}
	return nil
}

type CategoryListFilter struct {
	ParentID    *uuid.UUID
	IncludeTree bool
}

// GenerateSlug creates a URL-safe slug from a name.
// Polish diacritics are transliterated (ą→a, ł→l, etc.), the rest is lowercased and non-alphanum replaced with hyphens.
func GenerateSlug(name string) string {
	// Normalize to NFD, strip combining marks, then NFC
	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	result, _, _ := transform.String(t, name)

	// Polish-specific replacements that NFD doesn't handle
	replacer := strings.NewReplacer("ł", "l", "Ł", "L")
	result = replacer.Replace(result)

	result = strings.ToLower(result)

	// Replace non-alphanumeric with hyphens
	re := regexp.MustCompile(`[^a-z0-9]+`)
	result = re.ReplaceAllString(result, "-")

	return strings.Trim(result, "-")
}

// BuildCategoryTree builds a tree from a flat slice of categories.
// Categories must have ParentID set correctly. Root categories have nil ParentID.
func BuildCategoryTree(categories []ProductCategory) []ProductCategory {
	byParent := make(map[uuid.UUID][]ProductCategory)
	var roots []ProductCategory

	for _, c := range categories {
		c.Children = nil // reset
		if c.ParentID == nil {
			roots = append(roots, c)
		} else {
			byParent[*c.ParentID] = append(byParent[*c.ParentID], c)
		}
	}

	var attachChildren func(cats []ProductCategory) []ProductCategory
	attachChildren = func(cats []ProductCategory) []ProductCategory {
		for i := range cats {
			children := byParent[cats[i].ID]
			if len(children) > 0 {
				cats[i].Children = attachChildren(children)
			}
		}
		return cats
	}

	return attachChildren(roots)
}
