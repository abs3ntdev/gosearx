// types.go defines the richer result types beyond MainResult/Suggestion:
// answers (instant answers from plugins/answerers) and the finance types
// (quote, chart) that drive the interactive frontend.
package result

// Image is an image result (template images.html in SearXNG).
type Image struct {
	Type         Type   `json:"type"`
	EngineName   string `json:"engine"`
	Title        string `json:"title"`
	URL          string `json:"url"`          // page URL
	ImgSrc       string `json:"imgSrc"`       // full-size image
	ThumbnailSrc string `json:"thumbnailSrc"` // thumbnail
	Resolution   string `json:"resolution,omitempty"`
	Source       string `json:"source,omitempty"`
}

func (i *Image) Kind() Type     { return TypeImage }
func (i *Image) Engine() string { return i.EngineName }

// Video is a video result (template videos.html in SearXNG).
type Video struct {
	Type          Type   `json:"type"`
	EngineName    string `json:"engine"`
	Title         string `json:"title"`
	URL           string `json:"url"`
	Content       string `json:"content,omitempty"`
	Thumbnail     string `json:"thumbnail,omitempty"`
	Author        string `json:"author,omitempty"`
	Length        string `json:"length,omitempty"`
	PublishedDate string `json:"publishedDate,omitempty"`
}

func (v *Video) Kind() Type     { return TypeVideo }
func (v *Video) Engine() string { return v.EngineName }

// Answer is an instant answer shown above results (calculator, conversions, …).
type Answer struct {
	EngineName string `json:"engine"`
	Answer     string `json:"answer"`
	URL        string `json:"url,omitempty"`
}

func (a *Answer) Kind() Type     { return TypeAnswer }
func (a *Answer) Engine() string { return a.EngineName }

// InfoboxURL is a labeled link inside an infobox.
type InfoboxURL struct {
	Title string `json:"title"`
	URL   string `json:"url"`
}

// InfoboxAttr is a key/value attribute row inside an infobox.
type InfoboxAttr struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

// Infobox is a side-panel knowledge card (Wikipedia/Wikidata style).
type Infobox struct {
	Type       Type          `json:"type"`
	EngineName string        `json:"engine"`
	Title      string        `json:"title"`
	ID         string        `json:"id,omitempty"`
	Content    string        `json:"content,omitempty"`
	ImgSrc     string        `json:"imgSrc,omitempty"`
	URLs       []InfoboxURL  `json:"urls,omitempty"`
	Attributes []InfoboxAttr `json:"attributes,omitempty"`
}

func (i *Infobox) Kind() Type     { return TypeInfobox }
func (i *Infobox) Engine() string { return i.EngineName }

// Correction is a "did you mean" spelling correction.
type Correction struct {
	EngineName string `json:"engine"`
	Value      string `json:"correction"`
}

func (c *Correction) Kind() Type     { return TypeCorrection }
func (c *Correction) Engine() string { return c.EngineName }

// Quote is a financial quote: current price and change for a symbol.
type Quote struct {
	Type       Type    `json:"type"`
	EngineName string  `json:"engine"`
	Symbol     string  `json:"symbol"`
	Name       string  `json:"name,omitempty"`
	Currency   string  `json:"currency,omitempty"`
	Price      float64 `json:"price"`
	Change     float64 `json:"change,omitempty"`
	ChangePct  float64 `json:"changePct,omitempty"`
	URL        string  `json:"url,omitempty"`
}

func (q *Quote) Kind() Type     { return TypeQuote }
func (q *Quote) Engine() string { return q.EngineName }

// Candle is one OHLC data point in a chart series.
type Candle struct {
	T int64   `json:"t"` // unix seconds
	O float64 `json:"o"`
	H float64 `json:"h"`
	L float64 `json:"l"`
	C float64 `json:"c"`
	V float64 `json:"v,omitempty"`
}

// Chart is a time-series / candlestick chart result.
type Chart struct {
	Type       Type     `json:"type"`
	EngineName string   `json:"engine"`
	Title      string   `json:"title"`
	Symbol     string   `json:"symbol,omitempty"`
	Currency   string   `json:"currency,omitempty"`
	ChartKind  string   `json:"chartKind"` // "candlestick" | "line"
	Range      string   `json:"range,omitempty"`
	Series     []Candle `json:"series"`
	Quote      *Quote   `json:"quote,omitempty"`
	URL        string   `json:"url,omitempty"`
}

func (c *Chart) Kind() Type     { return TypeChart }
func (c *Chart) Engine() string { return c.EngineName }

// infoboxFromMap builds an Infobox from a generic map (Lua engine output).
func infoboxFromMap(engine string, m map[string]any) Result {
	ib := &Infobox{
		Type: TypeInfobox, EngineName: engine,
		Title:   str(m, "title"),
		ID:      str(m, "id"),
		Content: str(m, "content"),
		ImgSrc:  str(m, "imgSrc"),
	}
	if urls, ok := m["urls"].([]any); ok {
		for _, u := range urls {
			if um, ok := u.(map[string]any); ok {
				ib.URLs = append(ib.URLs, InfoboxURL{Title: str(um, "title"), URL: str(um, "url")})
			}
		}
	}
	if attrs, ok := m["attributes"].([]any); ok {
		for _, a := range attrs {
			if am, ok := a.(map[string]any); ok {
				ib.Attributes = append(ib.Attributes, InfoboxAttr{Label: str(am, "label"), Value: str(am, "value")})
			}
		}
	}
	if ib.Title == "" && ib.Content == "" {
		return nil
	}
	return ib
}

// videoFromMap builds a Video from a generic map.
func videoFromMap(engine string, m map[string]any) Result {
	v := &Video{
		Type: TypeVideo, EngineName: engine,
		Title:         str(m, "title"),
		URL:           str(m, "url"),
		Content:       str(m, "content"),
		Thumbnail:     str(m, "thumbnail"),
		Author:        str(m, "author"),
		Length:        str(m, "length"),
		PublishedDate: str(m, "publishedDate"),
	}
	if v.URL == "" && v.Title == "" {
		return nil
	}
	return v
}

// imageFromMap builds an Image from a generic map.
func imageFromMap(engine string, m map[string]any) Result {
	img := &Image{
		Type: TypeImage, EngineName: engine,
		Title:        str(m, "title"),
		URL:          str(m, "url"),
		ImgSrc:       str(m, "imgSrc"),
		ThumbnailSrc: str(m, "thumbnailSrc"),
		Resolution:   str(m, "resolution"),
		Source:       str(m, "source"),
	}
	if img.ThumbnailSrc == "" {
		img.ThumbnailSrc = img.ImgSrc
	}
	if img.URL == "" && img.ImgSrc == "" {
		return nil
	}
	return img
}

// quoteFromMap builds a Quote from a generic map (plugin/script output).
func quoteFromMap(engine string, m map[string]any) Result {
	return &Quote{
		Type: TypeQuote, EngineName: engine,
		Symbol:    str(m, "symbol"),
		Name:      str(m, "name"),
		Currency:  str(m, "currency"),
		Price:     num(m, "price"),
		Change:    num(m, "change"),
		ChangePct: num(m, "changePct"),
		URL:       str(m, "url"),
	}
}

// chartFromMap builds a Chart from a generic map.
func chartFromMap(engine string, m map[string]any) Result {
	c := &Chart{
		Type: TypeChart, EngineName: engine,
		Title:     str(m, "title"),
		Symbol:    str(m, "symbol"),
		Currency:  str(m, "currency"),
		ChartKind: str(m, "chartKind"),
		URL:       str(m, "url"),
	}
	if c.ChartKind == "" {
		c.ChartKind = "line"
	}
	if series, ok := m["series"].([]any); ok {
		for _, p := range series {
			if pm, ok := p.(map[string]any); ok {
				c.Series = append(c.Series, Candle{
					T: int64(num(pm, "t")), O: num(pm, "o"), H: num(pm, "h"),
					L: num(pm, "l"), C: num(pm, "c"), V: num(pm, "v"),
				})
			}
		}
	}
	return c
}
