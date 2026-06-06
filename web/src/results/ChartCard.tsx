import { useEffect, useRef, useState } from "react";
import {
  createChart,
  ColorType,
  type IChartApi,
  type CandlestickData,
  type LineData,
  type UTCTimestamp,
} from "lightweight-charts";
import {
  fetchChart,
  CHART_RANGES,
  RANGE_LABELS,
  type Chart,
  type ChartRange,
} from "../api";

// ChartCard renders an interactive finance chart with a Google-style range
// selector (1D 5D 1M 6M YTD 1Y 5Y MAX). Switching range refetches just the
// chart from /api/finance. Intraday ranges render as a line; longer ranges as
// candlesticks (matching the backend's chartKind).
export function ChartCard({ chart: initial }: { chart: Chart }): React.JSX.Element {
  const [chart, setChart] = useState<Chart>(initial);
  const [range, setRange] = useState<ChartRange>(
    (initial.range as ChartRange) ?? "1d",
  );
  const [loading, setLoading] = useState(false);
  const containerRef = useRef<HTMLDivElement>(null);
  const abortRef = useRef<AbortController | null>(null);

  // When the initial chart changes (new search), reset state.
  useEffect(() => {
    setChart(initial);
    setRange((initial.range as ChartRange) ?? "1d");
  }, [initial]);

  async function selectRange(r: ChartRange) {
    if (r === range || !chart.symbol) return;
    setRange(r);
    abortRef.current?.abort();
    const ac = new AbortController();
    abortRef.current = ac;
    setLoading(true);
    try {
      const next = await fetchChart(chart.symbol, r, ac.signal);
      setChart(next);
    } catch (err) {
      if ((err as Error).name !== "AbortError") {
        // keep previous chart on error
        console.error(err);
      }
    } finally {
      setLoading(false);
    }
  }

  // (Re)draw whenever the chart data changes.
  useEffect(() => {
    if (!containerRef.current) return;
    const c: IChartApi = createChart(containerRef.current, {
      height: 320,
      layout: {
        background: { type: ColorType.Solid, color: "transparent" },
        textColor: "#9aa0aa",
      },
      grid: {
        vertLines: { color: "#20242e" },
        horzLines: { color: "#20242e" },
      },
      rightPriceScale: { borderColor: "#20242e" },
      timeScale: {
        borderColor: "#20242e",
        timeVisible: chart.chartKind === "line",
        secondsVisible: false,
      },
      autoSize: true,
    });

    if (chart.chartKind === "line") {
      const data: LineData[] = chart.series.map((p) => ({
        time: p.t as UTCTimestamp,
        value: p.c,
      }));
      const up = (chart.quote?.changePct ?? 0) >= 0;
      const series = c.addLineSeries({
        color: up ? "#3ddc97" : "#ff6b6b",
        lineWidth: 2,
      });
      series.setData(data);
    } else {
      const data: CandlestickData[] = chart.series.map((p) => ({
        time: p.t as UTCTimestamp,
        open: p.o,
        high: p.h,
        low: p.l,
        close: p.c,
      }));
      const series = c.addCandlestickSeries({
        upColor: "#3ddc97",
        downColor: "#ff6b6b",
        borderVisible: false,
        wickUpColor: "#3ddc97",
        wickDownColor: "#ff6b6b",
      });
      series.setData(data);
    }
    c.timeScale().fitContent();
    return () => c.remove();
  }, [chart]);

  const q = chart.quote;
  const up = (q?.changePct ?? 0) >= 0;

  return (
    <section className="chart-card">
      <div className="chart-header">
        <div>
          <span className="chart-symbol">{chart.symbol}</span>
          {chart.title && chart.title !== chart.symbol && (
            <span className="chart-name"> · {chart.title}</span>
          )}
        </div>
        {q && (
          <div className="chart-quote">
            <span className="chart-price">
              {q.price.toLocaleString(undefined, { maximumFractionDigits: 4 })}{" "}
              {q.currency}
            </span>
            <span className={up ? "chart-change up" : "chart-change down"}>
              {up ? "▲" : "▼"} {Math.abs(q.changePct ?? 0).toFixed(2)}%
            </span>
          </div>
        )}
      </div>

      <div className="range-selector">
        {CHART_RANGES.map((r) => (
          <button
            key={r}
            className={r === range ? "range-btn active" : "range-btn"}
            onClick={() => selectRange(r)}
            disabled={loading}
          >
            {RANGE_LABELS[r]}
          </button>
        ))}
      </div>

      <div className={loading ? "chart-canvas loading" : "chart-canvas"} ref={containerRef} />
      <div className="chart-attr">via {chart.engine}</div>
    </section>
  );
}
