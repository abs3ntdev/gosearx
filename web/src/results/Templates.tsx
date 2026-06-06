// Rich result-type cards ported from SearXNG's result_templates: paper,
// torrent, map, video, code, file, keyvalue.
import type {
  Paper,
  Torrent,
  MapResult,
  VideoResult,
  CodeResult,
  FileResult,
  KeyValueResult,
} from "../api";

export function PaperCard({ paper }: { paper: Paper }): React.JSX.Element {
  return (
    <article className="result tmpl-paper">
      <h3 className="result-title">
        <a href={paper.url} target="_blank" rel="noreferrer">
          {paper.title}
        </a>
      </h3>
      <div className="paper-meta">
        {paper.authors && paper.authors.length > 0 && (
          <span className="paper-authors">{paper.authors.slice(0, 5).join(", ")}</span>
        )}
        {paper.journal && <span className="paper-journal">{paper.journal}</span>}
        {paper.publishedDate && <span className="paper-date">{paper.publishedDate}</span>}
      </div>
      {paper.content && <p className="result-content">{paper.content}</p>}
      <div className="result-meta">
        {paper.pdfUrl && (
          <a className="paper-pdf" href={paper.pdfUrl} target="_blank" rel="noreferrer">
            📄 PDF
          </a>
        )}
        {paper.doi && <span className="paper-doi">DOI: {paper.doi}</span>}
        <span className="badge">{paper.engine}</span>
      </div>
    </article>
  );
}

export function TorrentCard({ torrent }: { torrent: Torrent }): React.JSX.Element {
  return (
    <article className="result tmpl-torrent">
      <h3 className="result-title">
        <a href={torrent.url || torrent.magnetLink} target="_blank" rel="noreferrer">
          {torrent.title}
        </a>
      </h3>
      <div className="torrent-meta">
        <span className="torrent-seed">▲ {torrent.seeders} seeders</span>
        <span className="torrent-leech">▼ {torrent.leechers} leechers</span>
        {torrent.fileSize && <span className="torrent-size">{torrent.fileSize}</span>}
        {torrent.files ? <span>{torrent.files} files</span> : null}
      </div>
      <div className="result-meta">
        {torrent.magnetLink && (
          <a className="torrent-magnet" href={torrent.magnetLink}>
            🧲 Magnet
          </a>
        )}
        {torrent.torrentFile && (
          <a href={torrent.torrentFile} target="_blank" rel="noreferrer">
            ⬇ .torrent
          </a>
        )}
        <span className="badge">{torrent.engine}</span>
      </div>
    </article>
  );
}

export function MapCard({ place }: { place: MapResult }): React.JSX.Element {
  const hasCoords = !!(place.latitude && place.longitude);
  // OpenStreetMap embed centered on the coordinates.
  const bbox = hasCoords
    ? `${place.longitude! - 0.01}%2C${place.latitude! - 0.01}%2C${place.longitude! + 0.01}%2C${place.latitude! + 0.01}`
    : "";
  return (
    <article className="result tmpl-map">
      <div className="map-body">
        <h3 className="result-title">
          <a href={place.url} target="_blank" rel="noreferrer">
            {place.title}
          </a>
        </h3>
        {place.address && <p className="map-address">{place.address}</p>}
        {place.content && <p className="result-content">{place.content}</p>}
        <div className="result-meta">
          {hasCoords && (
            <span className="map-coords">
              {place.latitude!.toFixed(5)}, {place.longitude!.toFixed(5)}
            </span>
          )}
          <span className="badge">{place.engine}</span>
        </div>
      </div>
      {hasCoords && (
        <iframe
          className="map-embed"
          title={place.title}
          loading="lazy"
          src={`https://www.openstreetmap.org/export/embed.html?bbox=${bbox}&layer=mapnik&marker=${place.latitude}%2C${place.longitude}`}
        />
      )}
    </article>
  );
}

export function VideoCard({ video }: { video: VideoResult }): React.JSX.Element {
  const thumb = video.thumbnail
    ? `/image_proxy?url=${encodeURIComponent(video.thumbnail)}`
    : "";
  return (
    <article className={thumb ? "result tmpl-video has-thumb" : "result tmpl-video"}>
      {thumb && (
        <a className="video-thumb" href={video.url} target="_blank" rel="noreferrer">
          <img src={thumb} alt="" loading="lazy" />
          <span className="video-play">▶</span>
        </a>
      )}
      <div className="result-body">
        <h3 className="result-title">
          <a href={video.url} target="_blank" rel="noreferrer">
            {video.title}
          </a>
        </h3>
        <div className="video-meta">
          {video.author && <span>{video.author}</span>}
          {video.length && <span>{video.length}</span>}
          {video.publishedDate && <span>{video.publishedDate}</span>}
        </div>
        {video.content && <p className="result-content">{video.content}</p>}
        <span className="badge">{video.engine}</span>
      </div>
    </article>
  );
}

export function CodeCard2({ code }: { code: CodeResult }): React.JSX.Element {
  return (
    <article className="result tmpl-code">
      <h3 className="result-title">
        <a href={code.url} target="_blank" rel="noreferrer">
          {code.title}
        </a>
      </h3>
      {(code.repository || code.filename) && (
        <div className="result-meta">
          {code.repository && <span className="code-repo">{code.repository}</span>}
          {code.filename && <span className="code-file">{code.filename}</span>}
          {code.language && <span className="badge">{code.language}</span>}
        </div>
      )}
      {code.codeSnippet && (
        <pre className="gh-snippet">
          <code>{code.codeSnippet}</code>
        </pre>
      )}
      {code.content && !code.codeSnippet && <p className="result-content">{code.content}</p>}
    </article>
  );
}

export function FileCard({ file }: { file: FileResult }): React.JSX.Element {
  return (
    <article className="result tmpl-file">
      <h3 className="result-title">
        <a href={file.url} target="_blank" rel="noreferrer">
          {file.title}
        </a>
      </h3>
      <div className="result-meta">
        {file.fileType && <span className="badge">{file.fileType}</span>}
        {file.fileSize && <span>{file.fileSize}</span>}
        <span className="badge">{file.engine}</span>
      </div>
      {file.content && <p className="result-content">{file.content}</p>}
    </article>
  );
}

export function KeyValueCard({ kv }: { kv: KeyValueResult }): React.JSX.Element {
  return (
    <article className="result tmpl-kv">
      {kv.title && (
        <h3 className="result-title">
          {kv.url ? (
            <a href={kv.url} target="_blank" rel="noreferrer">
              {kv.title}
            </a>
          ) : (
            kv.title
          )}
        </h3>
      )}
      <table className="kv-table">
        <tbody>
          {Object.entries(kv.pairs).map(([k, v]) => (
            <tr key={k}>
              <td className="kv-key">{k}</td>
              <td className="kv-val">{v}</td>
            </tr>
          ))}
        </tbody>
      </table>
      <span className="badge">{kv.engine}</span>
    </article>
  );
}
