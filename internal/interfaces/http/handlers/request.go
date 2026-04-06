package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/pedrobarco/mroki/internal/application/commands"
	"github.com/pedrobarco/mroki/internal/application/queries"
	"github.com/pedrobarco/mroki/internal/domain/traffictesting"
	"github.com/pedrobarco/mroki/pkg/dto"
)

// Type aliases for backward compatibility

func CreateRequest(handler *commands.CreateRequestHandler) AppHandler {
	return func(w http.ResponseWriter, r *http.Request) error {
		var req dto.CreateRequestPayload

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			return dto.InvalidRequestBody(err)
		}

		gateIDStr := r.PathValue("gate_id")
		if gateIDStr == "" {
			return dto.MissingPathParam("gate_id")
		}

		// Build command
		cmd := commands.CreateRequestCommand{
			ID:        req.ID,
			GateID:    gateIDStr,
			Method:    req.Method,
			Path:      req.Path,
			Headers:   req.Headers,
			Body:      []byte(req.Body),
			CreatedAt: req.CreatedAt,
		}

		// Map responses
		for _, resp := range req.Responses {
			cmd.Responses = append(cmd.Responses, commands.CreateRequestResponseProps{
				ID:         resp.ID,
				Type:       resp.Type,
				StatusCode: resp.StatusCode,
				Headers:    resp.Headers,
				Body:       []byte(resp.Body),
				LatencyMs:  resp.LatencyMs,
				CreatedAt:  resp.CreatedAt,
			})
		}

		// Diff is optional — if provided by the proxy, pass it through;
		// otherwise the command handler computes it server-side
		if req.Diff != nil {
			cmd.Diff = &commands.CreateRequestDiffProps{
				Content: req.Diff.Content,
			}
		}

		// Execute command
		request, err := handler.Handle(r.Context(), cmd)
		if err != nil {
			// Map domain errors to HTTP status codes
			switch {
			case errors.Is(err, traffictesting.ErrInvalidGateID):
				return dto.InvalidGateID(gateIDStr)
			case errors.Is(err, traffictesting.ErrInvalidRequestID):
				return dto.InvalidRequestID(cmd.ID)
			default:
				return dto.NewError(
					http.StatusInternalServerError,
					dto.ErrorTypeInternalError,
					"Internal Server Error",
					"An unknown error occurred. Please try again later.",
					err,
				)
			}
		}

		resp := dto.Response[dto.Request]{
			Data: toRequestResponseDTO(request),
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			return dto.InvalidResponseBody(err)
		}

		return nil
	}
}

func GetRequestByID(handler *queries.GetRequestHandler) AppHandler {
	return func(w http.ResponseWriter, r *http.Request) error {
		gid := r.PathValue("gate_id")
		if gid == "" {
			return dto.MissingPathParam("gate_id")
		}

		rid := r.PathValue("request_id")
		if rid == "" {
			return dto.MissingPathParam("request_id")
		}

		query := queries.GetRequestQuery{
			ID:     rid,
			GateID: gid,
		}

		req, err := handler.Handle(r.Context(), query)
		if err != nil {
			switch {
			case errors.Is(err, traffictesting.ErrInvalidRequestID):
				return dto.InvalidRequestID(rid)
			case errors.Is(err, traffictesting.ErrInvalidGateID):
				return dto.InvalidGateID(gid)
			case errors.Is(err, traffictesting.ErrRequestNotFound):
				return dto.RequestNotFound(rid)
			case errors.Is(err, traffictesting.ErrGateNotFound):
				return dto.GateNotFound(gid)
			default:
				return dto.NewError(
					http.StatusInternalServerError,
					dto.ErrorTypeInternalError,
					"Internal Server Error",
					"An unknown error occurred. Please try again later.",
					err,
				)
			}
		}

		response := dto.Response[dto.RequestDetail]{
			Data: toFullRequestResponseDTO(req),
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(response); err != nil {
			return dto.InvalidResponseBody(err)
		}

		return nil
	}
}

func GetAllRequestsByGateID(handler *queries.ListRequestsHandler) AppHandler {
	return func(w http.ResponseWriter, r *http.Request) error {
		gid := r.PathValue("gate_id")
		if gid == "" {
			return dto.MissingPathParam("gate_id")
		}

		// Parse pagination query parameters
		limit, offset, err := parsePaginationQueryParams(r.URL.Query())
		if err != nil {
			return dto.InvalidRequestPagination(err)
		}

		// Parse filtering and sorting query parameters
		methods, pathPattern, fromDate, toDate, hasDiff, sortField, sortOrder, err := parseRequestQueryParams(r.URL.Query())
		if err != nil {
			return dto.InvalidRequestFilters(err)
		}

		query := queries.ListRequestsQuery{
			GateID:      gid,
			Limit:       limit,
			Offset:      offset,
			Methods:     methods,
			PathPattern: pathPattern,
			FromDate:    fromDate,
			ToDate:      toDate,
			HasDiff:     hasDiff,
			SortField:   sortField,
			SortOrder:   sortOrder,
		}

		result, err := handler.Handle(r.Context(), query)
		if err != nil {
			switch {
			case errors.Is(err, traffictesting.ErrInvalidGateID):
				return dto.InvalidGateID(gid)
			case errors.Is(err, traffictesting.ErrInvalidPagination):
				return dto.InvalidRequestPagination(err)
			case errors.Is(err, traffictesting.ErrInvalidFilters):
				return dto.InvalidRequestFilters(err)
			case errors.Is(err, traffictesting.ErrInvalidSort):
				return dto.InvalidRequestSort(err)
			default:
				return dto.NewError(
					http.StatusInternalServerError,
					dto.ErrorTypeInternalError,
					"Internal Server Error",
					"An unknown error occurred. Please try again later.",
					err,
				)
			}
		}

		// Map domain entities to DTOs
		data := make([]dto.Request, 0, len(result.Items))
		for _, req := range result.Items {
			data = append(data, toRequestResponseDTO(req))
		}

		// Use paginated response DTO
		response := dto.PaginatedResponse[[]dto.Request]{
			Data: data,
			Pagination: dto.PaginationMeta{
				Limit:   result.Limit,
				Offset:  result.Offset,
				Total:   result.Total,
				HasMore: result.HasMore,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(response); err != nil {
			return dto.InvalidResponseBody(err)
		}

		return nil
	}
}

func toRequestResponseDTO(req *traffictesting.Request) dto.Request {
	result := dto.Request{
		ID:        req.ID.String(),
		Method:    req.Method.String(),
		Path:      req.Path.String(),
		CreatedAt: req.CreatedAt,
		HasDiff:   !req.Diff.IsZero(),
	}

	for _, resp := range req.Responses {
		switch resp.Type {
		case traffictesting.ResponseTypeLive:
			result.LiveStatusCode = resp.StatusCode.Int()
			result.LiveLatencyMs = resp.LatencyMs
		case traffictesting.ResponseTypeShadow:
			result.ShadowStatusCode = resp.StatusCode.Int()
			result.ShadowLatencyMs = resp.LatencyMs
		}
	}

	return result
}

func toFullRequestResponseDTO(req *traffictesting.Request) dto.RequestDetail {
	result := dto.RequestDetail{
		ID:        req.ID.String(),
		Method:    req.Method.String(),
		Path:      req.Path.String(),
		CreatedAt: req.CreatedAt,
	}

	for _, resp := range req.Responses {
		result.Responses = append(result.Responses, dto.ResponseDetail{
			ID:         resp.ID.String(),
			Type:       string(resp.Type),
			StatusCode: resp.StatusCode.Int(),
			Headers:    resp.Headers.HTTPHeader(),
			Body:       string(resp.Body),
			LatencyMs:  resp.LatencyMs,
			CreatedAt:  resp.CreatedAt,
		})
	}

	result.Diff = dto.DiffDetail{
		Content: req.Diff.Content,
	}

	return result
}
