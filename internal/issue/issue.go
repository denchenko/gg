package issue

import (
	"bytes"
	"fmt"
	"regexp"
	"text/template"
)

// Issuer handles issue number extraction and URL generation.
type Issuer struct {
	urlTemplate *template.Template
	issueRegexp *regexp.Regexp
}

// NewIssuer creates a new Issuer instance with the given URL template.
func NewIssuer(urlTemplate string) *Issuer {
	var tmpl *template.Template
	if urlTemplate != "" {
		tmpl = template.Must(template.New("issueURL").Parse(urlTemplate))
	}

	return &Issuer{
		urlTemplate: tmpl,
		issueRegexp: regexp.MustCompile(`[A-Z]+\-[0-9]+`),
	}
}

// ExtractNumber extracts the first issue number from a merge request title.
func (i *Issuer) ExtractNumber(title string) string {
	matches := i.issueRegexp.FindString(title)

	return matches
}

// MakeURL generates an issue URL from a template and issue number.
func (i *Issuer) MakeURL(issueNumber string) (string, error) {
	if issueNumber == "" || i.urlTemplate == nil {
		return "", nil
	}

	var buf bytes.Buffer
	data := struct {
		Issue string
	}{
		Issue: issueNumber,
	}

	if err := i.urlTemplate.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute issue URL template: %w", err)
	}

	return buf.String(), nil
}
