import type { ImageResult, Quote } from "../api";

// QuoteCard renders a standalone finance quote (price + change) when an engine
// returns a quote without an accompanying chart.
export function QuoteCard({ quote }: { quote: Quote }): React.JSX.Element {
  const up = (quote.changePct ?? 0) >= 0;
  return (
    <section className="quote-card">
      <div>
        <span className="quote-symbol">{quote.symbol}</span>
        {quote.name && <span className="quote-name"> · {quote.name}</span>}
      </div>
      <div className="quote-figures">
        <span className="quote-price">
          {quote.price.toLocaleString(undefined, { maximumFractionDigits: 4 })} {quote.currency}
        </span>
        {quote.changePct != null && (
          <span className={up ? "chart-change up" : "chart-change down"}>
            {up ? "▲" : "▼"} {Math.abs(quote.changePct).toFixed(2)}%
          </span>
        )}
      </div>
    </section>
  );
}

// ImageGrid renders image results in a responsive grid.
export function ImageGrid({ images }: { images: ImageResult[] }): React.JSX.Element {
  return (
    <section className="image-grid">
      {images.map((img, i) => (
        <a
          key={i}
          className="image-cell"
          href={img.url || img.imgSrc}
          target="_blank"
          rel="noreferrer"
          title={img.title}
        >
          <img src={img.thumbnailSrc || img.imgSrc} alt={img.title} loading="lazy" />
        </a>
      ))}
    </section>
  );
}
