package tools

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/kubeshop/testkube/pkg/mcp/formatters"
)

// Insight tools expose the ingested granular insight series (performance/test
// metrics parsed from k6, jmeter, artillery, junit and influx reports, plus
// cross-tool canonical metrics) so an LLM can discover which metrics exist,
// query their values over time, and drill down to the executions behind them.
//
// All insight tools are scoped to the current MCP environment automatically;
// the environment is never a tool parameter.

// InsightSeriesCatalogParams filters the granular insight series catalog.
type InsightSeriesCatalogParams struct {
	Workflow        string
	Source          string
	MetricKey       string
	IdentityFilters string
	Query           string
	Page            int
	PageSize        int
}

// InsightMetricKeysParams filters the distinct granular insight metric keys.
type InsightMetricKeysParams struct {
	Workflow        string
	Source          string
	IdentityFilters string
	Query           string
	Page            int
	PageSize        int
}

// InsightMetricSeriesParams describes a granular insight time-series query.
type InsightMetricSeriesParams struct {
	Measure         string
	SeriesID        string
	Aggregate       string
	Segment         string
	Workflow        string
	IdentityFilters string
	Status          string
	TagFilter       string
	StartDate       string
	EndDate         string
}

// InsightExecutionsParams filters the executions behind an insight metric.
type InsightExecutionsParams struct {
	Measure         string
	IdentityFilters string
	Workflow        string
	Status          string
	TagFilter       string
	StartDate       string
	EndDate         string
	Page            int
	PageSize        int
}

// InsightSeriesLister lists the granular insight series catalog.
type InsightSeriesLister interface {
	ListInsightSeries(ctx context.Context, params InsightSeriesCatalogParams) (string, error)
}

// InsightMetricKeysLister lists the distinct granular insight metric keys.
type InsightMetricKeysLister interface {
	ListInsightMetricKeys(ctx context.Context, params InsightMetricKeysParams) (string, error)
}

// InsightMetricSeriesGetter returns a granular insight time series.
type InsightMetricSeriesGetter interface {
	GetInsightMetricSeries(ctx context.Context, params InsightMetricSeriesParams) (string, error)
}

// InsightExecutionsLister lists executions that produced an insight metric.
type InsightExecutionsLister interface {
	ListInsightExecutions(ctx context.Context, params InsightExecutionsParams) (string, error)
}

// ListInsightSeries creates a tool for discovering granular insight series.
func ListInsightSeries(client InsightSeriesLister) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	tool = mcp.NewTool("list_insight_series",
		mcp.WithDescription(ListInsightSeriesDescription),
		mcp.WithString("workflow", mcp.Description(InsightWorkflowDescription)),
		mcp.WithString("source", mcp.Description(InsightSourceDescription)),
		mcp.WithString("metricKey", mcp.Description(InsightMetricKeyDescription)),
		mcp.WithString("identityFilters", mcp.Description(InsightIdentityFiltersDescription)),
		mcp.WithString("q", mcp.Description(InsightQueryDescription)),
		mcp.WithNumber("page", mcp.Description(PageDescription)),
		mcp.WithNumber("pageSize", mcp.Description(PageSizeDescription)),
	)

	handler = func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		workflow, _ := OptionalParam[string](request, "workflow")
		source, _ := OptionalParam[string](request, "source")
		metricKey, _ := OptionalParam[string](request, "metricKey")
		identityFilters, _ := OptionalParam[string](request, "identityFilters")
		query, _ := OptionalParam[string](request, "q")
		page, err := OptionalIntParam(request, "page")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		if page < 0 {
			page = 0
		}
		pageSize, err := OptionalIntParam(request, "pageSize")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		if pageSize < 0 {
			pageSize = 0
		}
		result, err := client.ListInsightSeries(ctx, InsightSeriesCatalogParams{
			Workflow:        workflow,
			Source:          source,
			MetricKey:       metricKey,
			IdentityFilters: identityFilters,
			Query:           query,
			Page:            page,
			PageSize:        pageSize,
		})
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to list insight series: %v", err)), nil
		}

		formatted, err := formatters.FormatInsightSeriesCatalog(result)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to format insight series: %v", err)), nil
		}

		return mcp.NewToolResultText(formatted), nil
	}

	return tool, handler
}

// ListInsightMetricKeys creates a tool for listing distinct insight metric keys.
func ListInsightMetricKeys(client InsightMetricKeysLister) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	tool = mcp.NewTool("list_insight_metric_keys",
		mcp.WithDescription(ListInsightMetricKeysDescription),
		mcp.WithString("workflow", mcp.Description(InsightWorkflowDescription)),
		mcp.WithString("source", mcp.Description(InsightSourceDescription)),
		mcp.WithString("identityFilters", mcp.Description(InsightIdentityFiltersDescription)),
		mcp.WithString("q", mcp.Description(InsightQueryDescription)),
		mcp.WithNumber("page", mcp.Description(PageDescription)),
		mcp.WithNumber("pageSize", mcp.Description(PageSizeDescription)),
	)

	handler = func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		workflow, _ := OptionalParam[string](request, "workflow")
		source, _ := OptionalParam[string](request, "source")
		identityFilters, _ := OptionalParam[string](request, "identityFilters")
		query, _ := OptionalParam[string](request, "q")
		page, err := OptionalIntParam(request, "page")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		if page < 0 {
			page = 0
		}
		pageSize, err := OptionalIntParam(request, "pageSize")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		if pageSize < 0 {
			pageSize = 0
		}
		result, err := client.ListInsightMetricKeys(ctx, InsightMetricKeysParams{
			Workflow:        workflow,
			Source:          source,
			IdentityFilters: identityFilters,
			Query:           query,
			Page:            page,
			PageSize:        pageSize,
		})
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to list insight metric keys: %v", err)), nil
		}

		formatted, err := formatters.FormatInsightMetricKeys(result)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to format insight metric keys: %v", err)), nil
		}

		return mcp.NewToolResultText(formatted), nil
	}

	return tool, handler
}

// GetInsightMetricSeries creates a tool for querying a granular insight time series.
func GetInsightMetricSeries(client InsightMetricSeriesGetter) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	tool = mcp.NewTool("get_insight_metric_series",
		mcp.WithDescription(GetInsightMetricSeriesDescription),
		mcp.WithString("measure", mcp.Description(InsightMeasureDescription)),
		mcp.WithString("seriesId", mcp.Description(InsightSeriesIdDescription)),
		mcp.WithString("aggregate", mcp.Description(InsightAggregateDescription)),
		mcp.WithString("segment", mcp.Description(InsightSegmentDescription)),
		mcp.WithString("workflow", mcp.Description(InsightWorkflowDescription)),
		mcp.WithString("identityFilters", mcp.Description(InsightIdentityFiltersDescription)),
		mcp.WithString("status", mcp.Description(StatusDescription)),
		mcp.WithString("tagFilter", mcp.Description(InsightTagFilterDescription)),
		mcp.WithString("startDate", mcp.Description(StartDateDescription)),
		mcp.WithString("endDate", mcp.Description(EndDateDescription)),
		mcp.WithNumber("maxSamples", mcp.Description(InsightMaxSamplesDescription)),
	)

	handler = func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		measure, _ := OptionalParam[string](request, "measure")
		seriesID, _ := OptionalParam[string](request, "seriesId")
		if measure == "" && seriesID == "" {
			return mcp.NewToolResultError("either measure or seriesId is required"), nil
		}

		aggregate, _ := OptionalParam[string](request, "aggregate")
		if aggregate == "" {
			aggregate = "avg"
		}
		segment, _ := OptionalParam[string](request, "segment")
		workflow, _ := OptionalParam[string](request, "workflow")
		identityFilters, _ := OptionalParam[string](request, "identityFilters")
		status, _ := OptionalParam[string](request, "status")
		tagFilter, _ := OptionalParam[string](request, "tagFilter")
		startDate, _ := OptionalParam[string](request, "startDate")
		endDate, _ := OptionalParam[string](request, "endDate")
		maxSamples, err := OptionalIntParam(request, "maxSamples")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		result, err := client.GetInsightMetricSeries(ctx, InsightMetricSeriesParams{
			Measure:         measure,
			SeriesID:        seriesID,
			Aggregate:       aggregate,
			Segment:         segment,
			Workflow:        workflow,
			IdentityFilters: identityFilters,
			Status:          status,
			TagFilter:       tagFilter,
			StartDate:       startDate,
			EndDate:         endDate,
		})
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get insight metric series: %v", err)), nil
		}

		formatted, err := formatters.FormatInsightMetricSeries(result, maxSamples)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to format insight metric series: %v", err)), nil
		}

		return mcp.NewToolResultText(formatted), nil
	}

	return tool, handler
}

// ListInsightExecutions creates a tool for listing executions behind an insight metric.
func ListInsightExecutions(client InsightExecutionsLister) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	tool = mcp.NewTool("list_insight_executions",
		mcp.WithDescription(ListInsightExecutionsDescription),
		mcp.WithString("measure", mcp.Description(InsightMeasureDescription)),
		mcp.WithString("identityFilters", mcp.Description(InsightIdentityFiltersDescription)),
		mcp.WithString("workflow", mcp.Description(InsightWorkflowDescription)),
		mcp.WithString("status", mcp.Description(StatusDescription)),
		mcp.WithString("tagFilter", mcp.Description(InsightTagFilterDescription)),
		mcp.WithString("startDate", mcp.Description(StartDateDescription)),
		mcp.WithString("endDate", mcp.Description(EndDateDescription)),
		mcp.WithNumber("page", mcp.Description(PageDescription)),
		mcp.WithNumber("pageSize", mcp.Description(PageSizeDescription)),
	)

	handler = func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		measure, _ := OptionalParam[string](request, "measure")
		identityFilters, _ := OptionalParam[string](request, "identityFilters")
		workflow, _ := OptionalParam[string](request, "workflow")
		status, _ := OptionalParam[string](request, "status")
		tagFilter, _ := OptionalParam[string](request, "tagFilter")
		startDate, _ := OptionalParam[string](request, "startDate")
		endDate, _ := OptionalParam[string](request, "endDate")
		page, err := OptionalIntParam(request, "page")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		if page < 0 {
			page = 0
		}
		pageSize, err := OptionalIntParam(request, "pageSize")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		if pageSize < 0 {
			pageSize = 0
		}
		result, err := client.ListInsightExecutions(ctx, InsightExecutionsParams{
			Measure:         measure,
			IdentityFilters: identityFilters,
			Workflow:        workflow,
			Status:          status,
			TagFilter:       tagFilter,
			StartDate:       startDate,
			EndDate:         endDate,
			Page:            page,
			PageSize:        pageSize,
		})
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to list insight executions: %v", err)), nil
		}

		formatted, err := formatters.FormatInsightExecutions(result)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to format insight executions: %v", err)), nil
		}

		return mcp.NewToolResultText(formatted), nil
	}

	return tool, handler
}
