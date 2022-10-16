package dev

import (
	"fmt"
	"net/http"

	"code.gitea.io/gitea/core"
	bots_model "code.gitea.io/gitea/models/bots"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/web"
)

func BuildView(ctx *context.Context) {
	runID := ctx.ParamsInt64("runid")
	ctx.Data["RunID"] = runID
	jobID := ctx.ParamsInt64("jobid")
	if jobID <= 0 {
		runJobs, err := bots_model.GetRunJobsByRunID(ctx, runID)
		if err != nil {
			return
		}
		if len(runJobs) <= 0 {
			return
		}
		jobID = runJobs[0].ID
	}
	ctx.Data["JobID"] = jobID

	ctx.HTML(http.StatusOK, "dev/buildview")
}

type BuildViewRequest struct {
	StepLogCursors []struct {
		StepIndex int   `json:"stepIndex"`
		Cursor    int64 `json:"cursor"`
		Expanded  bool  `json:"expanded"`
	} `json:"stepLogCursors"`
}

type BuildViewResponse struct {
	StateData struct {
		BuildInfo struct {
			Title string `json:"title"`
		} `json:"buildInfo"`
		AllJobGroups   []BuildViewGroup `json:"allJobGroups"`
		CurrentJobInfo struct {
			Title  string `json:"title"`
			Detail string `json:"detail"`
		} `json:"currentJobInfo"`
		CurrentJobSteps []BuildViewJobStep `json:"currentJobSteps"`
	} `json:"stateData"`
	LogsData struct {
		StreamingLogs []BuildViewStepLog `json:"streamingLogs"`
	} `json:"logsData"`
}

type BuildViewGroup struct {
	Summary string          `json:"summary"`
	Jobs    []*BuildViewJob `json:"jobs"`
}

type BuildViewJob struct {
	Id     int64            `json:"id"`
	Name   string           `json:"name"`
	Status core.BuildStatus `json:"status"`
}

type BuildViewJobStep struct {
	Summary  string           `json:"summary"`
	Duration float64          `json:"duration"`
	Status   core.BuildStatus `json:"status"`
}

type BuildViewStepLog struct {
	StepIndex int                    `json:"stepIndex"`
	Cursor    int64                  `json:"cursor"`
	Lines     []BuildViewStepLogLine `json:"lines"`
}

type BuildViewStepLogLine struct {
	Ln int     `json:"ln"`
	M  string  `json:"m"`
	T  float64 `json:"t"`
}

func BuildViewPost(ctx *context.Context) {
	req := web.GetForm(ctx).(*BuildViewRequest)
	runID := ctx.ParamsInt64("runid")
	jobID := ctx.ParamsInt64("jobid")

	run, err := bots_model.GetRunByID(ctx, runID)
	if err != nil {
		if _, ok := err.(bots_model.ErrRunNotExist); ok {
			ctx.Error(http.StatusNotFound, err.Error())
			return
		}
		ctx.Error(http.StatusInternalServerError, err.Error())
		return
	}
	jobs, err := bots_model.GetRunJobsByRunID(ctx, run.ID)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, err.Error())
		return
	}

	var job *bots_model.RunJob
	if jobID != 0 {
		for _, v := range jobs {
			if v.ID == jobID {
				job = v
				break
			}
		}
		if job == nil {
			ctx.Error(http.StatusNotFound, fmt.Sprintf("run %v has no job %v", runID, jobID))
			return
		}
	}

	resp := &BuildViewResponse{}
	resp.StateData.BuildInfo.Title = run.Title

	respJobs := make([]*BuildViewJob, len(jobs))
	for i, v := range jobs {
		respJobs[i] = &BuildViewJob{
			Id:     v.ID,
			Name:   v.Name,
			Status: v.Status,
		}
	}

	resp.StateData.AllJobGroups = []BuildViewGroup{
		{
			Summary: "Only One Group", // TODO: maybe we don't need job group
			Jobs:    respJobs,
		},
	}

	if job != nil {
		var task *bots_model.Task
		if job.TaskID > 0 {
			task, err = bots_model.GetTaskByID(ctx, job.TaskID)
			if err != nil {
				ctx.Error(http.StatusInternalServerError, err.Error())
				return
			}
			task.Job = job
			if err := task.LoadAttributes(ctx); err != nil {
				ctx.Error(http.StatusInternalServerError, err.Error())
				return
			}
		}

		resp.StateData.CurrentJobInfo.Title = job.Name
		resp.LogsData.StreamingLogs = make([]BuildViewStepLog, 0, len(req.StepLogCursors))
		if job.TaskID == 0 {
			resp.StateData.CurrentJobInfo.Detail = "wait to be pick up by a runner"
		} else {
			resp.StateData.CurrentJobInfo.Detail = "TODO: more detail info" // TODO: more detail info

			steps := task.FullSteps()

			resp.StateData.CurrentJobSteps = make([]BuildViewJobStep, len(steps))
			for i, v := range steps {
				resp.StateData.CurrentJobSteps[i] = BuildViewJobStep{
					Summary:  v.Name,
					Duration: float64(v.Stopped - v.Started),
					Status:   core.StatusRunning, // TODO: add status to step,
				}
			}

			for _, cursor := range req.StepLogCursors {
				if cursor.Expanded {
					step := steps[cursor.StepIndex]
					var logRows []*bots_model.TaskLog
					if cursor.Cursor < step.LogLength || step.LogLength < 0 {
						logRows, err = bots_model.GetTaskLogs(task.ID, step.LogIndex+cursor.Cursor, step.LogLength-cursor.Cursor)
						if err != nil {
							ctx.Error(http.StatusInternalServerError, err.Error())
							return
						}
					}
					logLines := make([]BuildViewStepLogLine, len(logRows))
					for i, row := range logRows {
						logLines[i] = BuildViewStepLogLine{
							Ln: i,
							M:  row.Content,
							T:  float64(row.Timestamp),
						}
					}
					resp.LogsData.StreamingLogs = append(resp.LogsData.StreamingLogs, BuildViewStepLog{
						StepIndex: cursor.StepIndex,
						Cursor:    cursor.Cursor + int64(len(logLines)),
						Lines:     logLines,
					})
				}
			}
		}
	}

	ctx.JSON(http.StatusOK, resp)
}
