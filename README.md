# Taskmanager extension for Sliver
## Introduction
The Taskmanager extension for Sliver allows operators to interact with the
Windows Task Scheduler service. Because the Task Scheduler service is complex,
this extension supports a commonly used subset of tasks:

  - View all tasks on the system
  - Create a new task
  - Delete a task
  - Run a task

The extension primarily uses JSON for input and output of task information. JSON templates
can be generated for creating tasks. When generating a template, the values will be the defaults from the Task Scheduler service.
The JSON will contain the following properties:

  - `allow_demand_start` (default: `true`): Indicates that the task can be manually invoked using the `run` command.
  - `allow_hard_terminate` (default: `true`): Indicates that the task may be terminated by the Task Scheduler service using `TerminateProcess`
  - `dont_start_on_batteries` (default: `false`): Indicates that the task will not be started if the computer is running on battery power
  - `enabled` (default: `true`): Indicates whether the task is enabled
  - `hidden` (default: `false`): Indicates whether the window for the executable is hidden when the task executes
  - `idle_duration_hours`, `idle_duration_minutes`, `idle_duration_seconds` (default: 0, 10, 0): The amount of time the computer is idle before a task with an `idle` trigger will fire.
  - `priority` (default: 7): The priority level of the task. 0 is the highest, 10 is the lowest.
  - `restart_count` (default: 0): The number of times the Task Scheduler will attempt to restart the task.
  - `run_only_if_idle` (default: `false`): Indicates if the task will be run only when the computer is idle
  - `run_only_if_network_available` (default: `false`): Indicates that the task will run only when the network is available
  - `start_when_available` (default: `false`): Indicates if the task can be started at any time after its scheduled time has passed
  - `stop_if_going_on_batteries` (default: `false`): Indicates whether to stop the task if the computer is put on battery power
  - `stop_on_idle_end` (default: `true`): Terminate the task if the computer stops being idle, even if the task has not finished.
  - `time_limit_hours`, `time_limit_minutes`, `time_limit_seconds` (default: 72, 0, 0): The amount of time allowed to complete the task
  - `wake_to_run` (default: `false`): Wake the computer when the task is scheduled to run
  - `wait_timeout_hours`, `wait_timeout_minutes`, `wait_timeout_seconds` (default: 1, 0, 0): The amount of time that the Task Scheduler will wait for an idle condition to occur.

Tasks contain triggers that execute the task given specific conditions. Taskmanager supports
the following trigger types:

  - `boot`: Run the task when the machine boots. You must be an Administrator to schedule a task with this trigger.
  - `logon`: Run the task when a user logs on.
  - `idle`: Run the task when the user becomes idle.
  - `creation`: Run the task once when it is created.
  - `datetime`: Run the task once at a specific date and time.
  - `time_of_day`: Run the task daily at a specific time. Times are specified as `HH:MM` using the 24-hour clock.
  - `time_of_week`: Run the task on specific days of the week at a specific time. Days are specified as a comma separated list of numbers with 1 being Sunday and 7 being Saturday. For every day, use `*`.
  - `time_of_month`: Run the task on specific days of the month at a specific time. Days of the month are specified by their number, like 1 for the first. Use `*` for every day, and `last` for the last day of the month.

Triggers have some common properties:
  
  - `enabled`: `true` if the trigger is enabled, `false` if it is not
  - `delay`: The number of seconds to wait before firing the task. This does not apply to `idle` triggers. For `datetime`, `time_of_day`, `time_of_week`, and `time_of_month` triggers, this delay is a random amount of seconds that is added to the start time of the trigger.
  - `user`: The user to run the task as. A blank string is the current user, and a `*` denotes all users. To schedule tasks for other users, you
  must be an Administrator.
  - `time_limit`: The number of seconds that the task is allowed to execute.
  - `start_time`: A time when the task will start. Use this property to specify the datetime for a `datetime` task, the times for `time_of_day`, `time_of_week`, and `time_of_month` tasks.
  - `end_time`: The time that all occurances of this trigger will stop executing and the trigger will be disabled.

## Commands
Taskmanager accepts a string representing the action the operator wishes to take. If you would like JSON output to use for follow on
processing, call the extension with the `-j` or `--json` flag as the first flag (it must be the first flag and come before any commands).

If you are passing in a command that needs flags (like `-v` or `-o`) and you are using the
official Sliver client, you will need to run the command like this:
```bash
taskmanager -- view -v MyTask
```
or
```bash
# For JSON output
taskmanager -- -j view -v MyTask
```
Note the `--`. That tells the Sliver client to pass anything after `--` to the extension and not try to process that part.

If you need to pass strings, such as in the name of a task or a filepath, and you are using the official Sliver client, surround the string with two sets of quotes. For example, if you want to view a task named `My Task`, the command would be:
```bash
taskmanager view '"My Task"'
```

The syntax for these action strings is below.
### view
#### Syntax
```bash
# View all tasks
view

# View information about a specific task
view <task-path>

# Get a JSON representation of a task
view [--verbose/-v] <task-path>
```
The `view` command displays all tasks or a single task.

Without any arguments, the `view` command displays information about all registered tasks on the system
including the name of task, its path, whether it is enabled, the last and next run times, and its status.

Last and next run times are returned as RFC3339 timestamps in the local timezone.

When supplied with the path of one or more tasks, the `view` command will return the information described above
but only for the specified task(s). Tasks with spaces in the path must be enclosed in quotes. Multiple tasks must be specified
as a comma separated list.
Adding the `--verbose` or `-v` flag will return a JSON representation of the task that can be modified to create another task.
#### Examples
In the following examples, strings in quotes are surrounded by single quotes (`'`). This is only necessary when invoking `taskmanager` through the official Sliver client.
```
taskmanager view
Name                                                                            Path                                                                                                                         Enabled   Last Run                    Next Run                    Status     Execute                                                              

=============================================================================== ============================================================================================================================ ========= =========================== =========================== ========== ============================================================================================================================================================================================================================================================================
 .NET Framework NGEN v4.0.30319                                                  \Microsoft\Windows\.NET Framework\.NET Framework NGEN v4.0.30319                                                             yes       2024-02-08T07:46:48-08:00   1899-12-29T16:00:00-08:00   Ready      COM Class ID: {84F0FAE1-C27B-4F6F-807B-28CF6F96287D}, Data: /RuntimeWide
```
```json
taskmanager -j view
[{"name":"OneDrive Reporting Task-S-1-5-21-3900113992-3118352466-1278302697-1001","path":"\\OneDrive Report
ing Task-S-1-5-21-3900113992-3118352466-1278302697-1001","enabled":true,"lastRun":"2024-01-29T17:57:52-08:00","nextRun":"2024-01-30T17:57:52-08:00","status":"Ready","execute_actions":["%localappdata%\\Microsoft\\OneDrive\\OneDriveStandaloneUpdater.exe /reporting"]},...]
```
```json
taskmanager -j view '"Microsoft Compatibility Appraiser"'
[{"name":"Microsoft Compatibility Appraiser","path":"\\Microsoft\\Windows\\Application Experience\\Microsoft Compatibility Appraiser","enabled":true,"lastRun":"2024-01-29T20:22:28-08:00","nextRun":"2024-01-30T20:04:30-08:00","status":"Ready","execute_actions":["%windir%\\system32\\compattelrunner.exe "]}]
```
```json
taskmanager -j view '"Microsoft Compatibility Appraiser"',XblGameSaveTask
[{"name":"Microsoft Compatibility Appraiser","path":"\\Microsoft\\Windows\\Application Experience\\Microsoft Compatibility Appraiser","enabled":true,"lastRun":"2024-01-29T20:22:28-08:00","nextRun":"2024-01-30T19:24:35-08:00","status":"Ready","execute_actions":["%windir%\\system32\\compattelrunner.exe "]},{"name":"XblGameSaveTask","path":"\\Microsoft\\XblGameSave\\XblGameSaveTask","enabled":true,"lastRun":"1999-11-29T16:00:00-08:00","nextRun":"1899-12-29T16:00:00-08:00","status":"Ready","execute_actions":["%windir%\\System32\\XblGameSaveTask.exe standby"]}]
```
```
taskmanager view -v '"Microsoft Compatibility Appraiser"'
Microsoft Compatibility Appraiser (\Microsoft\Windows\Application Experience\Microsoft Compatibility Appraiser)
Last Run: 2024-02-08T19:03:48-08:00
Next Run: 2024-02-09T19:19:42-08:00
Executes: %windir%\system32\compattelrunner.exe

Task Definition:
{"allow_demand_start":true,"allow_hard_terminate":true,"dont_start_on_batteries":false,"enabled":true,"hidden":false,"idle_duration_hours":0,"idle_duration_minutes":0,"idle_duration_seconds":0,"wait_timeout_hours":0,"wait_timeout_minutes":0,"wait_timeout_seconds":0,"priority":7,"restart_count":0,"restart_on_idle":false,"run_only_if_idle":false,"run_only_if_network_available":true,"start_when_available":true,"stop_if_going_on_batteries":false,"stop_on_idle_end":true,"time_limit_hours":0,"time_limit_minutes":0,"time_limit_seconds":0,"wake_to_run":false,"triggers":[{"trigger_on":"datetime","enabled":true,"delay":0,"user":"","time_limit":0,"start_time":"2008-09-01T03:00:00","end_time":"0001-01-01T00:00:00"},{"trigger_on":"","enabled":false,"delay":0,"user":"","time_limit":0,"start_time":"0001-01-01T00:00:00","end_time":"0001-01-01T00:00:00"},{"trigger_on":"","enabled":false,"delay":0,"user":"","time_limit":0,"start_time":"0001-01-01T00:00:00","end_time":"0001-01-01T00:00:00"}]}
```
```json
taskmanager -j view -v "Microsoft Compatibility Appraiser"
[{"allow_demand_start":true,"allow_hard_terminate":true,"dont_start_on_batteries":false,"enabled":true,"hidden":false,"idle_duration_hours":0,"idle_duration_minutes":0,"idle_duration_seconds":0,"wait_timeout_hours":0,"wait_timeout_minutes":0,"wait_timeout_seconds":0,"priority":7,"restart_count":0,"restart_on_idle":false,"run_only_if_idle":false,"run_only_if_network_available":true,"start_when_available":true,"stop_if_going_on_batteries":false,"stop_on_idle_end":true,"time_limit_hours":0,"time_limit_minutes":0,"time_limit_seconds":0,"wake_to_run":false,"triggers":[{"trigger_on":"datetime","enabled":true,"delay":0,"user":"","time_limit":0,"start_time":"2008-09-01T03:00:00","end_time":"0001-01-01T00:00:00"},{"trigger_on":"","enabled":false,"delay":0,"user":"","time_limit":0,"start_time":"0001-01-01T00:00:00","end_time":"0001-01-01T00:00:00"},{"trigger_on":"","enabled":false,"delay":0,"user":"","time_limit":0,"start_time":"0001-01-01T00:00:00","end_time":"0001-01-01T00:00:00"}]}]
```
### view-folders
#### Syntax
```bash
view-folders
```
The `view-folders` command returns a list of folders registered with the Task Manager service.
#### Example
```
taskmanager view-folders
\
\Microsoft
\Microsoft\OneCore
...
```
```json
taskmanager -j view-folders

[{"path":"\\"},{"path":"\\Microsoft"},{"path":"\\Microsoft\\OneCore"},{"path":"\\Microsoft\\OneCore\\DirectX"},...]
```
### get-template
#### Syntax
```bash
get-template <comma separated list of trigger types>
```
The `get-template` command returns a template that can be used to fine tune the creation of a task.
#### Examples
```json
taskmanager get-template boot
{"allow_demand_start":true,"allow_hard_terminate":true,"dont_start_on_batteries":false,"enabled":true,"hidden":false,"idle_duration_hours":0,"idle_duration_minutes":10,"idle_duration_seconds":0,"wait_timeout_hours":1,"wait_timeout_minutes":0,"wait_timeout_seconds":0,"priority":7,"restart_count":0,"restart_on_idle":false,"run_only_if_idle":false,"run_only_if_network_available":false,"start_when_available":false,"stop_if_going_on_batteries":false,"stop_on_idle_end":true,"time_limit_hours":72,"time_limit_minutes":0,"time_limit_seconds":0,"wake_to_run":false,"triggers":[{"trigger_on":"boot","enabled":false,"delay":0,"user":"","time_limit":120,"start_time":"00:00","end_time":"00:00"}]}
```
```json
taskmanager get-template datetime,time_of_day
{"allow_demand_start":true,"allow_hard_terminate":true,"dont_start_on_batteries":false,"enabled":true,"hidden":false,"idle_duration_hours":0,"idle_duration_minutes":10,"idle_duration_seconds":0,"wait_timeout_hours":1,"wait_timeout_minutes":0,"wait_timeout_seconds":0,"priority":7,"restart_count":0,"restart_on_idle":false,"run_only_if_idle":false,"run_only_if_network_available":false,"start_when_available":false,"stop_if_going_on_batteries":false,"stop_on_idle_end":true,"time_limit_hours":72,"time_limit_minutes":0,"time_limit_seconds":0,"wake_to_run":false,"triggers":[{"trigger_on":"datetime","enabled":true,"delay":0,"user":"","time_limit":120,"start_time":"2006-01-02T15:04:05Z07:00","end_time":"00:00"},{"trigger_on":"time_of_day","enabled":true,"delay":0,"user":"","time_limit":120,"start_time":"00:00","end_time":"00:00","day_interval":1}]}
```
### create
#### Syntax
```bash
create [--overwrite/-o] <type_of_trigger> <trigger_arguments> <task_path_or_name> <command to execute> <command arguments>
```
The `create` command creates a new task on the system. It accepts the following types of triggers:

  - `custom`: This trigger type expects a JSON task generated either by `get-template` or `view <task_name>`. If you
  want to fine tune the parameters for a task or create a task with multiple triggers, this is the trigger type to use. Put your JSON in single quotes if you are using the offical Sliver client.
  - `boot`: Create a task that fires on boot. You must be part of the Administrator group to schedule a task with this trigger.
  This trigger does not take any trigger arguments.
  - `idle`: Create a task that executes when the user goes idle. This trigger does not take any trigger arguments.
  - `creation`: Create a task that executes when it is created. This trigger does not take any trigger arguments.
  - `login`: Creates a task that executes when the current user logs in. This trigger does not take any trigger arguments.
  - `once`: Creates a task that executes once at a specific date and time. The date and time must be specified in RFC3339 format
  (`YYYY-MM-DDTHH:MM:SS`). The time is interpreted to be local to the machine.
  - `daily`: Creates a task that fires once a day at a specific time. The time must be specified in `HH:MM` format (24 hour clock).
  The time is interpreted to be local to the mahcine.

If you need to overwrite an existing task, you must specify the `--overwrite` or `-o` flag. If you try to create a task with the same
name as a task that exists on the system and you do not specify the overwrite flag, you will get an error. If the executable has spaces in it, it must be enclosed in quotes. The arguments to the executable do not need to be enclosed in quotes.

Modifying a task is a three step process: get the representation of the task, modify parameters as necessary,
then call the `create` command with the overwrite flag to modify the task.
#### Examples
```bash
# Create a new task that executes notepad daily at 13:25
taskmanager create daily 13:25 MyTask '"C:\Windows\notepad.exe"'
```
```bash
# Create a new task that executes calc.exe on login
taskmanager create login MyCalc '"C:\Windows\System32\calc.exe"'
```
```bash
# Create a new task that executes an executable once on March 21, 2024 at 12:45
taskmanager create once 2024-03-21T12:45:00 MyDateTimeTask '"C:\Program Files\MyProgram\myprogram.exe"' -f -c 1
```
```json
# Create a new task that executes an program at 15:43 every Wednesday and Friday
create custom {"allow_demand_start":true,"allow_hard_terminate":true,"dont_start_on_batteries":false,"enabled":true,"hidden":false,"idle_duration_hours":0,"idle_duration_minutes":10,"idle_duration_seconds":0,"wait_timeout_hours":1,"wait_timeout_minutes":0,"wait_timeout_seconds":0,"priority":7,"restart_count":0,"restart_on_idle":false,"run_only_if_idle":false,"run_only_if_network_available":false,"start_when_available":false,"start_if_going_on_batteries":true,"stop_on_idle_end":true,"time_limit_hours":72,"time_limit_minutes":0,"time_limit_seconds":0,"wake_to_run":false,"triggers":[{"trigger_on":"time_of_week","enabled":true,"delay":0,"user":"","time_limit":120,"start_time":"15:43","end_time":"00:00","days_of_week":"4,6"}]} MyDateTimeTask "C:\Program Files\MyProgram\myprogram.exe" -f -c 1
```
### delete
#### Syntax
```bash
delete <task_path>
```
Delete the specified task by providing its path.
#### Examples
```bash
# Delete the task \MyTask (the leading \ is not necessary)
taskmanager delete MyTask
```
```bash
# Delete the task \Microsoft\XblGameSave\XblGameSaveTask
taskmanager delete \Microsoft\XblGameSave\XblGameSaveTask
```
### run
#### Syntax
```bash
run <task_path>
```
Run the specified task by providing its path.
#### Examples
```bash
# Run the task \MyTask (the leading \ is not necessary)
taskmanager run MyTask
```
```bash
# Run the task \Microsoft\XblGameSave\XblGameSaveTask
taskmanager run \Microsoft\XblGameSave\XblGameSaveTask
```