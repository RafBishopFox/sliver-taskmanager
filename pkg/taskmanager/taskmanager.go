package taskmanager

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/capnspacehook/taskmaster"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/rickb777/date/period"
)

const (
	successMessage  = `{"result": "success"}`
	RFC3339TimeNoTZ = "2006-01-02T15:04:05"
)

var (
	// Using the table style from Sliver for better integration into the official client
	SliverTableStyle = table.Style{
		Name: "SliverTable",
		Box: table.BoxStyle{
			BottomLeft:       " ",
			BottomRight:      " ",
			BottomSeparator:  " ",
			Left:             " ",
			LeftSeparator:    " ",
			MiddleHorizontal: "=",
			MiddleSeparator:  " ",
			MiddleVertical:   " ",
			PaddingLeft:      " ",
			PaddingRight:     " ",
			Right:            " ",
			RightSeparator:   " ",
			TopLeft:          " ",
			TopRight:         " ",
			TopSeparator:     " ",
			UnfinishedRow:    "~~",
		},
		Color: table.ColorOptions{
			IndexColumn:  text.Colors{},
			Footer:       text.Colors{},
			Header:       text.Colors{},
			Row:          text.Colors{},
			RowAlternate: text.Colors{},
		},
		Format: table.FormatOptions{
			Footer: text.FormatDefault,
			Header: text.FormatTitle,
			Row:    text.FormatDefault,
		},
		Options: table.Options{
			DrawBorder:      false,
			SeparateColumns: true,
			SeparateFooter:  false,
			SeparateHeader:  true,
			SeparateRows:    false,
		},
	}
)

/*
Splits a command, preserving strings in quotes
*/
func splitCommand(commandString string) []string {
	var parts []string
	var currentPart string
	var inQuotes bool

	for _, char := range commandString {
		switch {
		case char == ' ' && !inQuotes:
			if currentPart != "" {
				parts = append(parts, currentPart)
				currentPart = ""
			}
		case char == '"' || char == '\'':
			inQuotes = !inQuotes
			fallthrough
		default:
			currentPart += string(char)
		}
	}

	if currentPart != "" {
		parts = append(parts, currentPart)
	}

	return parts
}

/*
Takes a command string and breaks it up by quoted strings,
flags and their arguments, and positional arguments
*/
func parseCommand(commandString string) []string {
	var parsed []string
	var flag string

	parts := splitCommand(commandString)

	for _, part := range parts {
		if strings.HasPrefix(part, "-") {
			if flag != "" {
				parsed = append(parsed, flag)
			}
			flag = part
		} else if flag != "" {
			parsed = append(parsed, fmt.Sprintf("%s %s", flag, part))
			flag = ""
		} else {
			parsed = append(parsed, part)
		}
	}

	if flag != "" {
		parsed = append(parsed, flag)
	}

	return parsed
}

/*
Given a list of trigger types, returns a list of triggers that can be used
as templates
*/
func createTriggerTemplates(triggerTypes string) ([]Trigger, error) {
	var triggers []Trigger

	for _, triggerType := range strings.Split(triggerTypes, ",") {
		triggerType := strings.TrimSpace(triggerType)
		common := Trigger{
			TriggerOn: triggerType,
			Delay:     0,
			User:      "",
			TimeLimit: 120,
			StartTime: "00:00",
			EndTime:   "00:00",
			Enabled:   true,
		}
		switch triggerType {
		case BootTask, LogonTask, IdleTask, CreationTask:
			triggers = append(triggers, common)
		case TimeTask:
			common.StartTime = time.Now().Format(RFC3339TimeNoTZ)
			triggers = append(triggers, common)
		case DailyTask:
			common.DayInterval = 1
			triggers = append(triggers, common)
		case WeeklyTask:
			common.DaysOfWeek = "1,3,5"
			triggers = append(triggers, common)
		case MonthlyTask:
			common.DaysOfMonth = "1,7,11"
			common.MonthsOfYear = "2,4,6"
			common.RunOnLastWeekOfMonth = true
			triggers = append(triggers, common)
		default:
			return triggers, fmt.Errorf("%s is not a supported trigger", triggerType)
		}
	}

	return triggers, nil
}

// Converts a taskmaster trigger into a Trigger
func convertTrigger(trigger taskmaster.Trigger) (Trigger, error) {
	newTrigger := Trigger{
		StartTime: trigger.GetStartBoundary().Format(RFC3339TimeNoTZ),
		EndTime:   trigger.GetEndBoundary().Format(RFC3339TimeNoTZ),
		TimeLimit: uint(trigger.GetExecutionTimeLimit().Seconds()),
		Enabled:   trigger.GetEnabled(),
	}
	switch trigger.GetType() {
	// Nothing needed for idle
	// The type conversions should be fine, but going to check them anyway to avoid panics
	case taskmaster.TASK_TRIGGER_TIME:
		newTrigger.TriggerOn = TimeTask
		timeTrigger, ok := trigger.(taskmaster.TimeTrigger)
		if !ok {
			return newTrigger, fmt.Errorf("trigger conversion error")
		}
		newTrigger.Delay = uint(timeTrigger.RandomDelay.Seconds())
	case taskmaster.TASK_TRIGGER_DAILY:
		newTrigger.TriggerOn = DailyTask
		dailyTrigger, ok := trigger.(taskmaster.DailyTrigger)
		if !ok {
			return newTrigger, fmt.Errorf("trigger conversion error")
		}
		newTrigger.Delay = uint(dailyTrigger.RandomDelay.Seconds())
		newTrigger.DayInterval = uint(dailyTrigger.DayInterval)
	case taskmaster.TASK_TRIGGER_WEEKLY:
		newTrigger.TriggerOn = WeeklyTask
		weeklyTrigger, ok := trigger.(taskmaster.WeeklyTrigger)
		if !ok {
			return newTrigger, fmt.Errorf("trigger conversion error")
		}
		err := newTrigger.DaysOfWeekFromTrigger(weeklyTrigger.DaysOfWeek)
		if err != nil {
			return newTrigger, err
		}
		newTrigger.Delay = uint(weeklyTrigger.RandomDelay.Seconds())
	case taskmaster.TASK_TRIGGER_MONTHLY:
		newTrigger.TriggerOn = MonthlyTask
		monthlyTrigger, ok := trigger.(taskmaster.MonthlyTrigger)
		if !ok {
			return newTrigger, fmt.Errorf("trigger conversion error")
		}
		err := newTrigger.MonthsOfYearFromTrigger(monthlyTrigger.MonthsOfYear)
		if err != nil {
			return newTrigger, err
		}
		err = newTrigger.DaysOfMonthFromTrigger(monthlyTrigger.DaysOfMonth)
		if err != nil {
			return newTrigger, err
		}
		newTrigger.RunOnLastWeekOfMonth = monthlyTrigger.RunOnLastWeekOfMonth
	case taskmaster.TASK_TRIGGER_REGISTRATION:
		newTrigger.TriggerOn = CreationTask
		registrationTrigger, ok := trigger.(taskmaster.RegistrationTrigger)
		if !ok {
			return newTrigger, fmt.Errorf("trigger conversion error")
		}
		newTrigger.Delay = uint(registrationTrigger.Delay.Seconds())
	case taskmaster.TASK_TRIGGER_BOOT:
		newTrigger.TriggerOn = BootTask
		bootTrigger, ok := trigger.(taskmaster.BootTrigger)
		if !ok {
			return newTrigger, fmt.Errorf("trigger conversion error")
		}
		newTrigger.Delay = uint(bootTrigger.Delay.Seconds())
	case taskmaster.TASK_TRIGGER_LOGON:
		newTrigger.TriggerOn = LogonTask
		logonTrigger, ok := trigger.(taskmaster.LogonTrigger)
		if !ok {
			return newTrigger, fmt.Errorf("trigger conversion error")
		}
		if logonTrigger.UserID == "" {
			newTrigger.User = "*"
		} else {
			newTrigger.User = logonTrigger.UserID
		}
		newTrigger.Delay = uint(logonTrigger.Delay.Seconds())
	}

	return newTrigger, nil
}

// Converts a TaskMaster Definition to our TaskDefinition
func convertDefinitionToTaskDefinition(def taskmaster.Definition) (TaskDefinition, error) {
	td := TaskDefinition{
		AllowDemandStart:          def.Settings.AllowDemandStart,
		AllowHardTerminate:        def.Settings.AllowHardTerminate,
		DontStartOnBatteries:      def.Settings.DontStartOnBatteries,
		Enabled:                   def.Settings.Enabled,
		Hidden:                    def.Settings.Hidden,
		IdleDurationHours:         uint(def.Settings.IdleDuration.Hours()),
		IdleDurationMinutes:       uint(def.Settings.IdleDuration.Minutes()),
		IdleDurationSeconds:       uint(def.Settings.IdleDuration.Seconds()),
		WaitTimeoutHours:          uint(def.Settings.WaitTimeout.Hours()),
		WaitTimeoutMinutes:        uint(def.Settings.WaitTimeout.Minutes()),
		WaitTimeoutSeconds:        uint(def.Settings.WaitTimeout.Seconds()),
		Priority:                  def.Settings.Priority,
		RestartCount:              def.Settings.RestartCount,
		RestartOnIdle:             def.Settings.RestartOnIdle,
		RunOnlyIfIdle:             def.Settings.RunOnlyIfIdle,
		RunOnlyIfNetworkAvailable: def.Settings.RunOnlyIfNetworkAvailable,
		StartWhenAvailable:        def.Settings.StartWhenAvailable,
		StopIfGoingOnBatteries:    def.Settings.StopIfGoingOnBatteries,
		StopOnIdleEnd:             def.Settings.StopOnIdleEnd,
		TimeLimitHours:            uint(def.Settings.TimeLimit.Hours()),
		TimeLimitMinutes:          uint(def.Settings.TimeLimit.Minutes()),
		TimeLimitSeconds:          uint(def.Settings.TimeLimit.Seconds()),
		WakeToRun:                 def.Settings.WakeToRun,
		Triggers:                  []Trigger{},
	}

	for _, trigger := range def.Triggers {
		internalTrigger, err := convertTrigger(trigger)
		if err != nil {
			return td, err
		}
		td.Triggers = append(td.Triggers, internalTrigger)
	}

	return td, nil
}

// Quickly connect to the task manager service to figure out which user we are
func getCurrentUser() (string, error) {
	taskService, err := taskmaster.Connect()
	if err != nil {
		return "", err
	}
	defer taskService.Disconnect()

	return taskService.GetConnectedUser(), nil
}

// Returns a default taskmaster definition that we can build on
func createDefaultDefinition() *taskmaster.Definition {
	def := taskmaster.TaskService{}.NewTaskDefinition()
	return &def
}

// Adds Triggers to a taskmaster definition
func addTriggersToDefinition(def *taskmaster.Definition, triggers []Trigger) error {
	var err error

	for _, trigger := range triggers {
		// Convert each trigger to the associated trigger type
		switch trigger.TriggerOn {
		case BootTask:
			def.AddTrigger(taskmaster.BootTrigger{
				TaskTrigger: taskmaster.TaskTrigger{Enabled: trigger.Enabled},
				Delay:       period.NewHMS(0, 0, int(trigger.Delay)),
			})
		case LogonTask:
			var triggerUser string
			switch trigger.User {
			case "*":
				triggerUser = ""
			case "":
				triggerUser, err = getCurrentUser()
				if err != nil {
					return err
				}
			default:
				triggerUser = trigger.User
			}

			def.AddTrigger(taskmaster.LogonTrigger{
				TaskTrigger: taskmaster.TaskTrigger{Enabled: trigger.Enabled},
				Delay:       period.NewHMS(0, 0, int(trigger.Delay)),
				UserID:      triggerUser,
			})
		case IdleTask:
			def.AddTrigger(taskmaster.IdleTrigger{
				TaskTrigger: taskmaster.TaskTrigger{
					StartBoundary: time.Now(),
					Enabled:       trigger.Enabled,
				},
			})
		case CreationTask:
			def.AddTrigger(taskmaster.RegistrationTrigger{
				TaskTrigger: taskmaster.TaskTrigger{Enabled: trigger.Enabled},
				Delay:       period.NewHMS(0, 0, int(trigger.Delay)),
			})
		case TimeTask:
			startTime, err := time.Parse(RFC3339TimeNoTZ, trigger.StartTime)
			if err != nil {
				return err
			}
			/*
				Recreate the time as a local time
				using In(time.Local) would convert from UTC which would result in an incorrect time
			*/
			startTime = time.Date(startTime.Year(),
				startTime.Month(),
				startTime.Day(),
				startTime.Hour(),
				startTime.Minute(),
				startTime.Second(),
				startTime.Nanosecond(),
				time.Local)
			def.AddTrigger(taskmaster.TimeTrigger{
				TaskTrigger: taskmaster.TaskTrigger{
					StartBoundary: startTime,
					Enabled:       trigger.Enabled,
				},
				RandomDelay: period.NewHMS(0, 0, int(trigger.Delay)),
			})
		case DailyTask:
			startTime, err := time.Parse("15:04", trigger.StartTime)
			if err != nil {
				return err
			}
			startTime = time.Date(time.Now().Year(),
				time.Now().Month(),
				time.Now().Day(),
				startTime.Hour(),
				startTime.Minute(),
				0,
				0,
				time.Local)
			if trigger.DayInterval != 1 && trigger.DayInterval != 2 {
				return fmt.Errorf("currently only every day (1) or every other day (2) is supported for day interval")
			}
			def.AddTrigger(taskmaster.DailyTrigger{
				TaskTrigger: taskmaster.TaskTrigger{
					StartBoundary: startTime,
					Enabled:       trigger.Enabled,
				},
				DayInterval: taskmaster.DayInterval(trigger.DayInterval),
				RandomDelay: period.NewHMS(0, 0, int(trigger.Delay)),
			})
		case WeeklyTask:
			startTime, err := time.Parse("15:04", trigger.StartTime)
			if err != nil {
				return err
			}
			startTime = time.Date(time.Now().Year(),
				time.Now().Month(),
				time.Now().Day(),
				startTime.Hour(),
				startTime.Minute(),
				0,
				0,
				time.Local)
			var daysOfWeek taskmaster.DayOfWeek
			if trigger.DaysOfWeek == "*" {
				daysOfWeek = taskmaster.AllDays
			} else {
				daysOfWeek, err = trigger.ConvertDaysOfWeek()
				if err != nil {
					return err
				}
			}

			def.AddTrigger(taskmaster.WeeklyTrigger{
				TaskTrigger: taskmaster.TaskTrigger{
					StartBoundary: startTime,
					Enabled:       trigger.Enabled,
				},
				DaysOfWeek:   daysOfWeek,
				RandomDelay:  period.NewHMS(0, 0, int(trigger.Delay)),
				WeekInterval: taskmaster.EveryWeek,
			})
		case MonthlyTask:
			startTime, err := time.Parse("15:04", trigger.StartTime)
			if err != nil {
				return err
			}
			startTime = time.Date(time.Now().Year(),
				time.Now().Month(),
				time.Now().Day(),
				startTime.Hour(),
				startTime.Minute(),
				0,
				0,
				time.Local)
			var daysOfMonth taskmaster.DayOfMonth
			switch trigger.DaysOfMonth {
			case "last":
				daysOfMonth = taskmaster.LastDayOfMonth
			case "*":
				daysOfMonth = taskmaster.AllDaysOfMonth
			default:
				daysOfMonth, err = trigger.ConvertDaysOfMonth()
				if err != nil {
					return err
				}
			}
			var months taskmaster.Month
			if trigger.MonthsOfYear == "*" {
				months = taskmaster.AllMonths
			} else {
				months, err = trigger.ConvertMonths()
				if err != nil {
					return err
				}
			}

			def.AddTrigger(taskmaster.MonthlyTrigger{
				TaskTrigger: taskmaster.TaskTrigger{
					StartBoundary: startTime,
					Enabled:       trigger.Enabled,
				},
				DaysOfMonth:          daysOfMonth,
				MonthsOfYear:         months,
				RandomDelay:          period.NewHMS(0, 0, int(trigger.Delay)),
				RunOnLastWeekOfMonth: trigger.RunOnLastWeekOfMonth,
			})
		}
	}

	return nil
}

// Converts a TaskDefinition to a taskmaster definition
func convertTaskDefinitionToDefinition(def TaskDefinition) (*taskmaster.Definition, error) {
	var err error = nil

	newDefinition := taskmaster.TaskService{}.NewTaskDefinition()
	newDefinition.Settings.AllowDemandStart = def.AllowDemandStart
	newDefinition.Settings.AllowHardTerminate = def.AllowHardTerminate
	newDefinition.Settings.DontStartOnBatteries = def.DontStartOnBatteries
	newDefinition.Settings.Enabled = def.Enabled
	newDefinition.Settings.Hidden = def.Hidden
	newDefinition.Settings.IdleSettings.IdleDuration = period.NewHMS(
		int(def.IdleDurationHours),
		int(def.IdleDurationMinutes),
		int(def.IdleDurationSeconds),
	)
	newDefinition.Settings.IdleSettings.WaitTimeout = period.NewHMS(
		int(def.WaitTimeoutHours),
		int(def.WaitTimeoutMinutes),
		int(def.WaitTimeoutSeconds),
	)
	newDefinition.Settings.Priority = def.Priority
	newDefinition.Settings.RestartCount = def.RestartCount
	newDefinition.Settings.RestartOnIdle = def.RestartOnIdle
	newDefinition.Settings.RunOnlyIfIdle = def.RunOnlyIfIdle
	newDefinition.Settings.RunOnlyIfNetworkAvailable = def.RunOnlyIfNetworkAvailable
	newDefinition.Settings.StartWhenAvailable = def.StartWhenAvailable
	newDefinition.Settings.StopIfGoingOnBatteries = def.StopIfGoingOnBatteries
	newDefinition.Settings.StopOnIdleEnd = def.StopOnIdleEnd
	newDefinition.Settings.TimeLimit = period.NewHMS(
		int(def.TimeLimitHours),
		int(def.TimeLimitMinutes),
		int(def.TimeLimitSeconds),
	)
	newDefinition.Settings.WakeToRun = def.WakeToRun

	err = addTriggersToDefinition(&newDefinition, def.Triggers)

	return &newDefinition, err
}

// Build a template for a given list of trigger types
func getTemplate(triggerTypes string) (string, error) {
	taskService := taskmaster.TaskService{}
	triggers, err := createTriggerTemplates(triggerTypes)
	if err != nil {
		return "", err
	}
	newDef := taskService.NewTaskDefinition()
	newDef.Settings.DontStartOnBatteries = false
	newDef.Settings.StopIfGoingOnBatteries = false
	taskDef, err := convertDefinitionToTaskDefinition(newDef)
	if err != nil {
		return "", err
	}
	taskDef.Triggers = triggers
	result, err := json.Marshal(taskDef)
	if err != nil {
		return "", err
	}
	return string(result), err
}

// Get a deduplicated list of folders for a given folder
func getSubFolders(folder *taskmaster.TaskFolder) []FolderInfo {
	var folders []FolderInfo

	folders = append(folders, FolderInfo{Path: folder.Path})
	for _, innerFolder := range folder.SubFolders {
		// We need to be sure not to include duplicates
		for _, subFolder := range getSubFolders(innerFolder) {
			if !slices.Contains(folders, FolderInfo{Path: subFolder.Path}) {
				folders = append(folders, FolderInfo{Path: subFolder.Path})
			}
		}
	}

	return folders
}

// Get a list of task folders registered with the task manager service
func viewFolders(jsonOutput bool) (string, error) {
	var folders []FolderInfo
	taskService, err := taskmaster.Connect()
	if err != nil {
		return "", err
	}
	defer taskService.Disconnect()

	allFolders, err := taskService.GetTaskFolders()
	if err != nil {
		return "", err
	}
	folders = getSubFolders(&allFolders)

	if jsonOutput {
		jsonResult, err := json.Marshal(folders)
		if err != nil {
			return "", err
		}
		return string(jsonResult), nil
	}

	result := ""
	for _, folder := range folders {
		result += fmt.Sprintf("%s\n", folder.Path)
	}

	return result, nil
}

/*
Get a list of all tasks or a single task by name.
If verbose is true, a JSON string representing the task will be returned.
This string can be used as a template to modify or duplicate the task.
*/
func viewTasks(filter string, verbose bool, jsonOutput bool) (string, error) {
	var err error

	taskService, err := taskmaster.Connect()
	if err != nil {
		return "", err
	}
	defer taskService.Disconnect()

	// Get all registered tasks
	allTasks, err := taskService.GetRegisteredTasks()
	if err != nil {
		return "", err
	}
	defer allTasks.Release()

	var filterParts []string
	if filter == "" {
		filterParts = nil
	} else {
		filterParts = strings.Split(filter, ",")
		// Strip quotes
		for idx, filter := range filterParts {
			filterStripped := filter
			filterStripped = strings.TrimLeft(filterStripped, "\"")
			filterStripped = strings.TrimRight(filterStripped, "\"")
			filterStripped = strings.ReplaceAll(filterStripped, "/", "\\")
			filterParts[idx] = filterStripped
		}
	}

	var tasks []TaskInfo
	var verboseTasks []TaskDefinition

	for _, task := range allTasks {
		filterMatch := false
		taskActions := []string{}

		if filterParts != nil {
			// Check for a name match
			var pathInTasks bool
			var nameInTasks bool
			for _, filter := range filterParts {
				pathInTasks = false
				nameInTasks = false

				// Check if this is a path
				if !strings.HasPrefix(filter, "\\") {
					pathInTasks = "\\"+filter == task.Path
				} else {
					pathInTasks = filter == task.Path
				}
				nameInTasks = filter == task.Name
				if pathInTasks || nameInTasks {
					filterMatch = true
					break
				}
			}
		} else {
			filterMatch = true
		}

		if !filterMatch {
			continue
		}

		if verbose {
			// Verbose is only supported for a specific task / tasks, so print this task as a definition JSON
			taskDef, err := convertDefinitionToTaskDefinition(task.Definition)
			if err != nil {
				return "", err
			}
			verboseTasks = append(verboseTasks, taskDef)
		}

		for _, action := range task.Definition.Actions {
			switch action.GetType() {
			case taskmaster.TASK_ACTION_EXEC:
				execAction, ok := action.(taskmaster.ExecAction)
				if !ok {
					continue
				}
				if execAction.Args == "" {
					taskActions = append(taskActions, execAction.Path)
				} else {
					taskActions = append(taskActions, fmt.Sprintf("%s %s", execAction.Path, execAction.Args))
				}

			case taskmaster.TASK_ACTION_COM_HANDLER:
				comAction, ok := action.(taskmaster.ComHandlerAction)
				if !ok {
					continue
				}
				taskActions = append(taskActions, fmt.Sprintf("COM Class ID: %s, Data: %s", comAction.ClassID, comAction.Data))
			default:
				// TASK_ACTION_SHOW_MESSAGE and TASK_ACTION_SEND_EMAIL do not appear to be implemented
				continue
			}
		}

		tasks = append(tasks, TaskInfo{
			Name:    task.Name,
			Path:    task.Path,
			Enabled: task.Enabled,
			LastRun: task.LastRunTime.Format(RFC3339TimeNoTZ),
			NextRun: task.NextRunTime.Format(RFC3339TimeNoTZ),
			Status:  task.State.String(),
			Actions: taskActions,
		})
	}

	if len(tasks) == 0 && len(verboseTasks) == 0 {
		if filterParts != nil {
			return "", fmt.Errorf("could not find tasks matching the provided filter")
		} else {
			return "", fmt.Errorf("could not find any tasks registered on the system")
		}
	}

	if jsonOutput {
		var jsonResult []byte
		if verbose {
			jsonResult, err = json.Marshal(verboseTasks)
		} else {
			jsonResult, err = json.Marshal(tasks)
		}

		if err != nil {
			return "", err
		}

		return string(jsonResult), nil
	}

	result := ""
	if verbose {
		for idx, verboseTask := range verboseTasks {
			jsonResult, err := json.Marshal(verboseTask)
			if err != nil {
				return "", err
			}
			task := tasks[idx]
			result += fmt.Sprintf("%s (%s)\n", task.Name, task.Path)
			result += fmt.Sprintf("Last Run: %s\n", task.LastRun)
			result += fmt.Sprintf("Next Run: %s\n", task.NextRun)
			result += fmt.Sprintf("Executes: %s\n\n", strings.Join(task.Actions, ", "))
			result += fmt.Sprintf("Task Definition:\n%s\n\n", string(jsonResult))
		}
	} else {
		tw := table.NewWriter()
		tw.SetStyle(SliverTableStyle)
		tw.AppendHeader(table.Row{
			"Name",
			"Path",
			"Enabled",
			"Last Run",
			"Next Run",
			"Status",
			"Execute",
		})
		tw.SortBy([]table.SortBy{
			{Number: 1, Mode: table.Asc},
		})
		for _, task := range tasks {
			enabled := "yes"
			if !task.Enabled {
				enabled = "no"
			}
			tw.AppendRow(table.Row{
				task.Name,
				task.Path,
				enabled,
				task.LastRun,
				task.NextRun,
				task.Status,
				strings.Join(task.Actions, ", "),
			})
		}
		result = tw.Render()
	}

	return result, nil
}

/*
Create a task. There are subcommands for the different supported tasks:
- custom: Takes in a JSON task specification (generated by get-definition or from the details of an existing task)
- daily: accepts a time that a given task will run every day
- once: accepts an RFC3339 datetime without timezone for a task that will only occur once
- boot: Schedule a task that starts on boot (requires admin privileges)
- login: Schedule a task that starts when a user logs in
- idle: Schedule a task when the user's session becomes idle
- creation: Schedule a task that fires one time when it is created

All subcommands expect a task path (or name) and the command to run (with the command's arguments)
*/
func createTask(args []string, jsonOutput bool) (string, error) {
	overwrite := false
	taskDef := TaskDefinition{}
	var def *taskmaster.Definition
	var command string

	/*
		For all options, there will be an optional flag (--overwrite/-o)
		This flag must come as before the rest of command
	*/
	if strings.HasPrefix(args[0], "--overwrite") || strings.HasPrefix(args[0], "-o") {
		overwrite = true
		command = strings.TrimPrefix(args[0], "--overwrite ")
		command = strings.TrimPrefix(command, "-o ")
	} else {
		command = args[0]
	}

	/*
		Validate the second argument which is the timing
		Must be one of: custom, daily, once, boot, login, idle, creation
		The first three expect an argument after, the last four do not take an argument
	*/
	switch command {
	case "custom":
		// Try to read ahead and make a task definition from the provided JSON
		// We need at least 3 arguments total (the timing type, the definition JSON, a path/name, and an executable)
		if len(args) >= 4 {
			err := json.Unmarshal([]byte(args[1]), &taskDef)
			if err != nil {
				return "", err
			}
			def, err = convertTaskDefinitionToDefinition(taskDef)
			if err != nil {
				return "", err
			}
			args = args[2:]
		} else {
			return "", fmt.Errorf("not enough arguments provided")
		}
	case "daily":
		// Try to read ahead and make a task definition using the provided time
		// We need at least 3 arguments total (the timing type, the time of day to execute, a path/name, and an executable)
		if len(args) >= 4 {
			def = createDefaultDefinition()
			err := addTriggersToDefinition(def, []Trigger{{
				TriggerOn:   DailyTask,
				StartTime:   args[1],
				DayInterval: 1,
			}})
			if err != nil {
				return "", err
			}
			args = args[2:]
		} else {
			return "", fmt.Errorf("not enough arguments provided")
		}
	case "once":
		// Try to read ahead and make a task definition using the provided date/time
		// We need at least 3 arguments total (the timing type, the datetime to execute, a path/name, and an executable)
		if len(args) >= 4 {
			def = createDefaultDefinition()
			err := addTriggersToDefinition(def, []Trigger{{
				TriggerOn: TimeTask,
				StartTime: args[1],
			}})
			if err != nil {
				return "", err
			}
			args = args[2:]
		} else {
			return "", fmt.Errorf("not enough arguments provided")
		}
	case "boot":
		// Make sure we have an executable and path defined
		if len(args) >= 3 {
			def = createDefaultDefinition()
			err := addTriggersToDefinition(def, []Trigger{{
				TriggerOn: BootTask,
			}})
			if err != nil {
				return "", err
			}
			args = args[1:]
		} else {
			return "", fmt.Errorf("not enough arguments provided")
		}
	case "login":
		// Make sure we have an executable and path defined
		if len(args) >= 3 {
			def = createDefaultDefinition()
			err := addTriggersToDefinition(def, []Trigger{{
				TriggerOn: LogonTask,
			}})
			if err != nil {
				return "", err
			}
			args = args[1:]
		} else {
			return "", fmt.Errorf("not enough arguments provided")
		}
	case "idle":
		// Make sure we have an executable and path defined
		if len(args) >= 3 {
			def = createDefaultDefinition()
			err := addTriggersToDefinition(def, []Trigger{{
				TriggerOn: IdleTask,
			}})
			if err != nil {
				return "", err
			}
			args = args[1:]
		} else {
			return "", fmt.Errorf("not enough arguments provided")
		}
	case "creation":
		// Make sure we have an executable and path defined
		if len(args) >= 3 {
			def = createDefaultDefinition()
			err := addTriggersToDefinition(def, []Trigger{{
				TriggerOn: CreationTask,
			}})
			if err != nil {
				return "", err
			}
			args = args[1:]
		} else {
			return "", fmt.Errorf("not enough arguments provided")
		}
	default:
		return "", fmt.Errorf("%s is not a supported task timing type", args[0])
	}

	// The path of the task is next
	taskPath := args[0]
	// Remove quotes around the path if they exist
	taskPath = strings.TrimLeft(taskPath, "\"")
	taskPath = strings.TrimRight(taskPath, "\"")

	if !strings.HasPrefix(taskPath, "\\") {
		taskPath = "\\" + taskPath
	}
	args = args[1:]

	// Create an action for the executable and add it to the definition
	execArgs := strings.Join(args[1:], " ")

	execAction := taskmaster.ExecAction{
		Path: args[0],
		Args: execArgs,
	}
	def.AddAction(execAction)

	// Register (create) the task
	// Connect to the Task Scheduler service
	taskService, err := taskmaster.Connect()
	if err != nil {
		return "", err
	}
	defer taskService.Disconnect()

	// We do not need the task back. We just need to make sure it gets registered
	_, registered, err := taskService.CreateTask(taskPath, *def, overwrite)
	if err != nil {
		return "", err
	}
	if !registered && !overwrite {
		return "", fmt.Errorf("task exists, but overwrite was not specified")
	}

	if jsonOutput {
		return successMessage, nil
	} else {
		return fmt.Sprintf("Successfully created task %s", taskPath), nil
	}
}

// Delete a task
func deleteTask(taskPath string) error {
	// Connect to the Task Scheduler service
	taskService, err := taskmaster.Connect()
	if err != nil {
		return err
	}
	defer taskService.Disconnect()

	// Remove quotes around the path if they exist
	taskPath = strings.TrimLeft(taskPath, "\"")
	taskPath = strings.TrimRight(taskPath, "\"")

	if !strings.HasPrefix(taskPath, "\\") {
		taskPath = "\\" + taskPath
	}
	taskPath = strings.ReplaceAll(taskPath, "/", "\\")

	return taskService.DeleteTask(taskPath)
}

// Run a task
func runTask(taskPath string) error {
	// Connect to the Task Scheduler service
	taskService, err := taskmaster.Connect()
	if err != nil {
		return err
	}
	defer taskService.Disconnect()

	// Remove quotes around the path if they exist
	taskPath = strings.TrimLeft(taskPath, "\"")
	taskPath = strings.TrimRight(taskPath, "\"")

	if !strings.HasPrefix(taskPath, "\\") {
		taskPath = "\\" + taskPath
	}
	taskPath = strings.ReplaceAll(taskPath, "/", "\\")

	// Get the task
	task, err := taskService.GetRegisteredTask(taskPath)
	if err != nil {
		return err
	}

	// Run the task - we do not need the running task back
	_, err = task.Run()
	return err
}

// Do stuff
func ExecuteCommand(args string) (result string, err error) {

	command := parseCommand(args)
	if len(command) == 0 {
		// Then we got an empty command string
		return "", fmt.Errorf("a command is required")
	}

	jsonOutput := false
	if strings.HasPrefix(command[0], "-j") || strings.HasPrefix(command[0], "--json") {
		jsonOutput = true
		command[0] = strings.TrimPrefix(command[0], "-j ")
		command[0] = strings.TrimPrefix(command[0], "--json ")
	}

	// The command is the first element in the slice
	switch command[0] {
	case "view":
		if len(command) > 1 {
			// View accepts up to two arguments: an optional --verbose/-v flag and the name of the specific task to get info about
			if strings.HasPrefix(command[1], "-v") || strings.HasPrefix(command[1], "--verbose") {
				taskName := command[1]
				taskName = strings.TrimPrefix(taskName, "-v ")
				taskName = strings.TrimPrefix(taskName, "--verbose ")
				result, err = viewTasks(taskName, true, jsonOutput)
			} else {
				result, err = viewTasks(command[1], false, jsonOutput)
			}
		} else {
			result, err = viewTasks("", false, jsonOutput)
		}
	case "view-folders":
		result, err = viewFolders(jsonOutput)
	case "get-template":
		if len(command) > 1 {
			result, err = getTemplate(command[1])
		} else {
			err = fmt.Errorf("get-template requires a list of triggers")
		}
	case "create":
		if len(command) > 1 {
			result, err = createTask(command[1:], jsonOutput)
		} else {
			err = fmt.Errorf("not enough arguments")
		}
	case "delete":
		if len(command) > 1 {
			err = deleteTask(command[1])
			if err == nil {
				if jsonOutput {
					result = successMessage
				} else {
					result = fmt.Sprintf("Successfully deleted %s", command[1])
				}
			}
		} else {
			err = fmt.Errorf("not enough arguments")
		}
	case "run":
		if len(command) > 1 {
			err = runTask(command[1])
			if err == nil {
				if jsonOutput {
					result = successMessage
				} else {
					result = fmt.Sprintf("Successfully ran task %s", command[1])
				}
			}
		} else {
			err = fmt.Errorf("not enough arguments")
		}
	default:
		err = fmt.Errorf("command %s is not supported", command[0])
	}
	return
}
