package controller

import (
	"net/http"

	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/application"
	"github.com/almighty/almighty-core/criteria"
	"github.com/almighty/almighty-core/errors"
	"github.com/almighty/almighty-core/jsonapi"
	query "github.com/almighty/almighty-core/query/simple"
	"github.com/almighty/almighty-core/workitem"

	"github.com/goadesign/goa"
	errs "github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
)

// PlannerBacklogController implements the planner_backlog resource.
type PlannerBacklogController struct {
	*goa.Controller
	db application.DB
}

// NewPlannerBacklogController creates a planner_backlog controller.
func NewPlannerBacklogController(service *goa.Service, db application.DB) *PlannerBacklogController {
	return &PlannerBacklogController{Controller: service.NewController("PlannerBacklogController"), db: db}
}

func (c *PlannerBacklogController) List(ctx *app.ListPlannerBacklogContext) error {
	spaceID, err := uuid.FromString(ctx.ID)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, goa.ErrNotFound(err.Error()))
	}

	offset, limit := computePagingLimts(ctx.PageOffset, ctx.PageLimit)

	exp, err := query.Parse(ctx.Filter)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, errors.NewBadParameterError("could not parse filter", err))
	}
	if ctx.FilterAssignee != nil {
		exp = criteria.And(exp, criteria.Equals(criteria.Field("system.assignees"), criteria.Literal([]string{*ctx.FilterAssignee})))
	}
	if ctx.FilterWorkitemtype != nil {
		exp = criteria.And(exp, criteria.Equals(criteria.Field("Type"), criteria.Literal([]uuid.UUID{*ctx.FilterWorkitemtype})))
	}
	if ctx.FilterArea != nil {
		exp = criteria.And(exp, criteria.Equals(criteria.Field(workitem.SystemArea), criteria.Literal(string(*ctx.FilterArea))))
	}

	exp = criteria.Not(criteria.Field(workitem.SystemState), criteria.Literal(workitem.SystemStateClosed))

	return application.Transactional(c.db, func(appl application.Application) error {
		var result []*workitem.WorkItem
		// Get the root iteration
		iteration, err := appl.Iterations().RootIteration(ctx.Context, spaceID)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, errs.Wrap(err, "unable to fetch root iteration"))
		}
		exp = criteria.Equals(criteria.Field(workitem.SystemIteration), criteria.Literal(iteration.ID.String()))

		// Get the list of work item types that derive of PlannerItem in the space
		var expWits criteria.Expression
		wits, err := appl.WorkItemTypes().ListPlannerItems(ctx.Context, spaceID)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, errs.Wrap(err, "unable to fetch work item types that derives of planner item"))
		}
		if len(wits) >= 1 {
			expWits = criteria.Equals(criteria.Field("Type"), criteria.Literal(wits[0].ID.String()))
			for _, wit := range wits[1:] {
				witIDStr := wit.ID.String()
				expWits = criteria.Or(expWits, criteria.Equals(criteria.Field("Type"), criteria.Literal(witIDStr)))
			}
			exp = criteria.And(exp, expWits)
		} else {
			// If there isn't work item types, return an empty array.
			return ctx.OK(&app.WorkItemList{
				Data:  []*app.WorkItem{},
				Links: &app.PagingLinks{},
				Meta:  &app.WorkItemListResponseMeta{TotalCount: 0},
			})
		}

		// Get the list of work items for the following criteria
		result, tc, err := appl.WorkItems().List(ctx.Context, spaceID, exp, &offset, &limit)
		count := int(tc)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, errs.Wrap(err, "error listing backlog items"))
		}

		lastMod := findLastModified(result)
		if ifMod, ok := ctx.RequestData.Header["If-Modified-Since"]; ok {
			ifModSince, err := http.ParseTime(ifMod[0])
			if err == nil {
				if lastMod.Before(ifModSince) || lastMod.Equal(ifModSince) {
					return ctx.NotModified()
				}
			}
		}
		response := app.WorkItemList{
			Data:  ConvertWorkItems(ctx.RequestData, result),
			Links: &app.PagingLinks{},
			Meta:  &app.WorkItemListResponseMeta{TotalCount: count},
		}

		setPagingLinks(response.Links, buildAbsoluteURL(ctx.RequestData), len(result), offset, limit, count)

		ctx.ResponseData.Header().Set("Last-Modified", lastModifiedTime(lastMod))
		return ctx.OK(&response)
	})
}