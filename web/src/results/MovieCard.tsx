// MovieCard renders a Google-style knowledge panel for a film or TV show:
// poster, trailer link, ratings (IMDb/RT/Metacritic/TMDB), where-to-watch,
// and top cast. Populated by the internal/media service (TMDB + OMDb).
import type { Movie, MovieProvider } from "../api";

function ratingClass(source: string): string {
  switch (source) {
    case "IMDb":
      return "rt-imdb";
    case "Rotten Tomatoes":
      return "rt-rt";
    case "Metacritic":
      return "rt-meta";
    default:
      return "rt-tmdb";
  }
}

const PROVIDER_GROUPS: { type: MovieProvider["type"]; label: string }[] = [
  { type: "stream", label: "Stream" },
  { type: "free", label: "Free" },
  { type: "ads", label: "With ads" },
  { type: "rent", label: "Rent" },
  { type: "buy", label: "Buy" },
];

export function MovieCard({ movie }: { movie: Movie }): React.JSX.Element {
  const meta = [
    movie.year,
    movie.mediaType === "tv"
      ? movie.seasons
        ? `${movie.seasons} season${movie.seasons > 1 ? "s" : ""}`
        : "TV series"
      : "Movie",
    movie.runtime,
    movie.genres?.slice(0, 3).join(", "),
  ].filter(Boolean);

  return (
    <section className="movie-card">
      {movie.backdrop && (
        <div className="movie-backdrop" style={{ backgroundImage: `url(${movie.backdrop})` }} />
      )}
      <div className="movie-body">
        {movie.poster && (
          <img className="movie-poster" src={movie.poster} alt={movie.title} loading="lazy" />
        )}
        <div className="movie-main">
          <h2 className="movie-title">
            {movie.title}
            {movie.year && <span className="movie-year"> ({movie.year})</span>}
          </h2>
          <div className="movie-meta">{meta.join(" · ")}</div>
          {movie.tagline && <p className="movie-tagline">{movie.tagline}</p>}

          {movie.ratings && movie.ratings.length > 0 && (
            <div className="movie-ratings">
              {movie.ratings.map((r) => (
                <span className={`movie-rating ${ratingClass(r.source)}`} key={r.source}>
                  <span className="rt-src">{r.source}</span>
                  <span className="rt-val">{r.value}</span>
                </span>
              ))}
            </div>
          )}

          {movie.overview && <p className="movie-overview">{movie.overview}</p>}

          {movie.directors && movie.directors.length > 0 && (
            <div className="movie-credit">
              <span className="movie-credit-label">
                {movie.mediaType === "tv" ? "Creator" : "Director"}
                {movie.directors.length > 1 ? "s" : ""}:
              </span>{" "}
              {movie.directors.join(", ")}
            </div>
          )}

          <div className="movie-actions">
            {movie.trailerUrl && (
              <a className="movie-btn primary" href={movie.trailerUrl} target="_blank" rel="noreferrer">
                ▶ Trailer
              </a>
            )}
            {movie.imdbUrl && (
              <a className="movie-btn" href={movie.imdbUrl} target="_blank" rel="noreferrer">
                IMDb
              </a>
            )}
            {movie.url && (
              <a className="movie-btn" href={movie.url} target="_blank" rel="noreferrer">
                TMDB
              </a>
            )}
            {movie.justWatchUrl && (
              <a className="movie-btn" href={movie.justWatchUrl} target="_blank" rel="noreferrer">
                Where to watch
              </a>
            )}
          </div>
        </div>
      </div>

      {movie.providers && movie.providers.length > 0 && (
        <div className="movie-watch">
          {PROVIDER_GROUPS.map((g) => {
            const items = movie.providers!.filter((p) => p.type === g.type);
            if (items.length === 0) return null;
            return (
              <div className="movie-watch-row" key={g.type}>
                <span className="movie-watch-label">
                  {g.label}
                  {movie.providerRegion ? ` (${movie.providerRegion})` : ""}
                </span>
                <div className="movie-providers">
                  {items.map((p) => (
                    <span className="movie-provider" key={p.name} title={p.name}>
                      {p.logo ? (
                        <img src={p.logo} alt={p.name} loading="lazy" />
                      ) : (
                        <span className="movie-provider-text">{p.name}</span>
                      )}
                    </span>
                  ))}
                </div>
              </div>
            );
          })}
        </div>
      )}

      {movie.cast && movie.cast.length > 0 && (
        <div className="movie-cast">
          <div className="movie-cast-title">Cast</div>
          <div className="movie-cast-row">
            {movie.cast.map((c) => (
              <div className="movie-castmember" key={c.name + c.character}>
                {c.photo ? (
                  <img src={c.photo} alt={c.name} loading="lazy" />
                ) : (
                  <div className="movie-cast-noimg">{c.name.charAt(0)}</div>
                )}
                <div className="movie-cast-name">{c.name}</div>
                {c.character && <div className="movie-cast-char">{c.character}</div>}
              </div>
            ))}
          </div>
        </div>
      )}
    </section>
  );
}
