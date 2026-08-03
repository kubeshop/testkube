package tools

import (
	"context"
	"fmt"
	"strconv"

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
		mcp.WithString("page", mcp.Description(PageDescription)),
		mcp.WithString("pageSize", mcp.Description(PageSizeDescription)),
	)

	handler = func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		workflow, err := OptionalParam[string](request, "workflow")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		source, err := OptionalParam[string](request, "source")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		metricKey, err := OptionalParam[string](request, "metricKey")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		identityFilters, err := OptionalParam[string](request, "identityFilters")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		query, err := OptionalParam[string](request, "q")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		page := 0
		if pageStr := request.GetString("page", "0"); pageStr != "" {
			if p, err := strconv.Atoi(pageStr); err == nil && p >= 0 {
				page = p
			}
		}
		pageSize := 10
		if pageSizeStr := request.GetString("pageSize", "10"); pageSizeStr != "" {
			if ps, err := strconv.Atoi(pageSizeStr); err == nil && ps > 0 {
				pageSize = ps
			}
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
		mcp.WithString("page", mcp.Description(PageDescription)),
		mcp.WithString("pageSize", mcp.Description(PageSizeDescription)),
	)

	handler = func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		workflow, err := OptionalParam[string](request, "workflow")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		source, err := OptionalParam[string](request, "source")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		identityFilters, err := OptionalParam[string](request, "identityFilters")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		query, err := OptionalParam[string](request, "q")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		page := 0
		if pageStr := request.GetString("page", "0"); pageStr != "" {
			if p, err := strconv.Atoi(pageStr); err == nil && p >= 0 {
				page = p
			}
		}
		pageSize := 10
		if pageSizeStr := request.GetString("pageSize", "10"); pageSizeStr != "" {
			if ps, err := strconv.Atoi(pageSizeStr); err == nil && ps > 0 {
				pageSize = ps
			}
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
		measure, err := OptionalParam[string](request, "measure")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		seriesID, err := OptionalParam[string](request, "seriesId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		if seriesID != "" {
			measure = ""
		}
		if measure == "" && seriesID == "" {
			return mcp.NewToolResultError("either measure or seriesId is required"), nil
		}

		aggregate, err := OptionalParam[string](request, "aggregate")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		if aggregate == "" {
			aggregate = "avg"
		}
		segment, err := OptionalParam[string](request, "segment")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		workflow, err := OptionalParam[string](request, "workflow")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		identityFilters, err := OptionalParam[string](request, "identityFilters")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		status, err := OptionalParam[string](request, "status")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		tagFilter, err := OptionalParam[string](request, "tagFilter")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		startDate, err := OptionalParam[string](request, "startDate")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		endDate, err := OptionalParam[string](request, "endDate")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		maxSamples, err := OptionalIntParam(request, "maxSamples")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		if maxSamples < 0 {
			maxSamples = 0
		} else if maxSamples > 500 {
			maxSamples = 500
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
		mcp.WithString("page", mcp.Description(PageDescription)),
		mcp.WithString("pageSize", mcp.Description(PageSizeDescription)),
	)

	handler = func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		measure, err := OptionalParam[string](request, "measure")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		identityFilters, err := OptionalParam[string](request, "identityFilters")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		workflow, err := OptionalParam[string](request, "workflow")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		status, err := OptionalParam[string](request, "status")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		tagFilter, err := OptionalParam[string](request, "tagFilter")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		startDate, err := OptionalParam[string](request, "startDate")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		endDate, err := OptionalParam[string](request, "endDate")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		page := 0
		if pageStr := request.GetString("page", "0"); pageStr != "" {
			if p, err := strconv.Atoi(pageStr); err == nil && p >= 0 {
				page = p
			}
		}
		pageSize := 10
		if pageSizeStr := request.GetString("pageSize", "10"); pageSizeStr != "" {
			if ps, err := strconv.Atoi(pageSizeStr); err == nil && ps > 0 {
				pageSize = ps
			}
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
