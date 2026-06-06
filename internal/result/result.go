// Package result defines the typed result schema produced by engines and
// consumed by the frontend. It is the Go successor to SearXNG's
// searx/result_types package (msgspec structs) and result_templates/*.html.
//
// The design is a discriminated union: every result carries a Type, and the
// frontend has a registry mapping Type -> React component. Adding a new display
// type = add a struct here + a component in the web app.
package result

// Type is the discriminator for a result. It mirrors SearXNG's "template"
// concept (default.html, images.html, ...) but is an explicit enum.
type Type string

const (
	TypeMain       Type = "main"       // generic text result (default.html)
	TypeImage      Type = "image"      // images.html
	TypeVideo      Type = "video"      // videos.html
	TypeAnswer     Type = "answer"     // answer/*.html
	TypeSuggestion Type = "suggestion" // search suggestions
	TypeCorrection Type = "correction" // "did you mean"
	TypeInfobox    Type = "infobox"    // infobox panel

	// New rich types beyond SearXNG's text/image capabilities.
	TypeQuote Type = "quote" // finance: price + change
	TypeChart Type = "chart" // time-series / candlestick chart
)

// Result is implemented by every concrete result type. Kind() returns the
// discriminator so the merge/score layer and the JSON encoder can dispatch.
type Result interface {
	Kind() Type
	// Engine reports which engine produced the result (for scoring & display).
	Engine() string
}

// MainResult is the standard text/web result (SearXNG MainResult / default.html).
type MainResult struct {
	Type          Type     `json:"type"`
	EngineName    string   `json:"engine"`
	Title         string   `json:"title"`
	URL           string   `json:"url"`
	Content       string   `json:"content,omitempty"`
	Thumbnail     string   `json:"thumbnail,omitempty"`
	PublishedDate string   `json:"publishedDate,omitempty"`
	Engines       []string `json:"engines,omitempty"`
	Positions     []int    `json:"-"`
	Score         float64  `json:"score,omitempty"`
}

func (r *MainResult) Kind() Type     { return TypeMain }
func (r *MainResult) Engine() string { return r.EngineName }

// Suggestion is a search suggestion string.
type Suggestion struct {
	EngineName string `json:"engine"`
	Value      string `json:"suggestion"`
}

func (s *Suggestion) Kind() Type     { return TypeSuggestion }
func (s *Suggestion) Engine() string { return s.EngineName }

// EngineResults is what an engine's Response returns: a flat list of results.
// It is the Go analogue of searx.result_types.EngineResults.
type EngineResults []Result
