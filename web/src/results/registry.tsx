// The result-type registry is the extensibility mechanism for the frontend:
// it maps a result's `type` to the React component that renders it. This is the
// successor to SearXNG's result_templates/*.html — but typed and composable.
//
// To add a new display type (e.g. a finance chart): create a component and
// register it here under the matching ResultType. No other code changes needed.

import type { JSX } from "react";
import type { MainResult, ResultType } from "../api";
import { MainResultCard } from "./MainResultCard";

export type ResultComponent = (props: {
  result: MainResult;
  query: string;
}) => JSX.Element;

const registry: Partial<Record<ResultType, ResultComponent>> = {
  main: MainResultCard,
};

// renderResult picks the component for a result's type, falling back to the
// generic main card so unknown types still display. query is passed for
// term highlighting.
export function renderResult(result: MainResult, key: number, query: string): JSX.Element {
  const Component = registry[result.type] ?? MainResultCard;
  return <Component result={result} query={query} key={key} />;
}
