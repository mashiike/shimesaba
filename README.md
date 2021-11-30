![Latest GitHub release](https://img.shields.io/github/release/mashiike/shimesaba.svg)
![Github Actions test](https://github.com/mashiike/shimesaba/workflows/Test/badge.svg?branch=main)
[![Go Report Card](https://goreportcard.com/badge/mashiike/shimesaba)](https://goreportcard.com/report/mashiike/shimesaba) [![License](https://img.shields.io/badge/license-MIT-blue.svg)](https://github.com/mashiike/shimesaba/blob/master/LICENSE)
# shimesaba

For SRE to operate and monitor services using Mackerel.
## Description

shimesaba is a tool for tracking SLO/ErrorBudget using Mackerel as an SLI measurement service.

1. Get and aggregate Mackerel (host/service) metrics within the calculated period.
2. Calculate the SLI from the metric obtained in step 1 and determine if it is an SLO violation in the rolling window.
3. Calculate the time (minutes) of SLO violation within the time frame of the rolling window and calculate the error budget.
4. Post the calculated error budget, failure time for SLO violation, etc. as Mackerel service metric.


## Install

### binary packages

[Releases](https://github.com/mashiike/shimesaba/releases).

### Homebrew tap

```console
$ brew install mashiike/tap/shimesaba
```

## Usage

### as CLI command

```console
$ shimesaba -config config.yaml -mackerel-apikey <Mackerel API Key> run
```

```console
NAME:
   shimesaba - A command line tool for tracking SLO/ErrorBudget using Mackerel as an SLI measurement service.

USAGE:
   shimesaba [global options] command [command options] [arguments...]

VERSION:
   current

COMMANDS:
   dashboard  manage mackerel dashboard for SLI/SLO
   run        run shimesaba. this is main feature
   help, h    Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --config value, -c value           config file path, can set multiple [$CONFIG, $SHIMESABA_CONFIG]
   --debug                            output debug log (default: false) [$SHIMESABA_DEBUG]
   --mackerel-apikey value, -k value  for access mackerel API (default: *********) [$MACKEREL_APIKEY, $SHIMESABA_MACKEREL_APIKEY]
   --help, -h                         show help (default: false)
   --version, -v                      print the version (default: false)
2021/11/14 23:29:45 [error] Required flag "config" not set
```

run command usage is follow
```console
$ shimesaba run --help
NAME:
   main run - run shimesaba. this is main feature

USAGE:
   shimesaba -config <config file> run [command options]

OPTIONS:
   --dry-run         report output stdout and not put mackerel (default: false) [$SHIMESABA_DRY_RUN]
   --backfill value  generate report before n point (default: 3) [$BACKFILL, $SHIMESABA_BACKFILL]
   --help, -h        show help (default: false)
```

### as AWS Lambda function

`shimesaba` binary also runs as AWS Lambda function. 
shimesaba implicitly behaves as a run command when run as a bootstrap with a Lambda Function


CLI options can be specified from environment variables. For example, when `MACKEREL_APIKEY` environment variable is set, the value is set to `-mackerel-apikey` option.

Example Lambda functions configuration.

```json
{
  "FunctionName": "shimesaba",
  "Environment": {
    "Variables": {
      "SHIMESABA_CONFIG": "config.yaml",
      "MACKEREL_APIKEY": "<Mackerel API KEY>"
    }
  },
  "Handler": "shimesaba",
  "MemorySize": 128,
  "Role": "arn:aws:iam::0123456789012:role/lambda-function",
  "Runtime": "provided.al2",
  "Timeout": 300
}
```

### Configuration file

YAML format.

```yaml
required_version: ">=0.0.0"

metrics:
  - id: alb_p90_response_time
    name: custom.alb.response.time_p90
    type: host
    service_name: prod
    host_name: dummy-alb
    aggregation_interval: 1m
    aggregation_method: max
  - id: component_response_time
    name: component.dummy.response_time
    type: service
    service_name: prod
    aggregation_interval: 1m
    aggregation_method: avg

definitions:
  - id: latency
    service_name: prod 
    time_frame: 28d #4weeks
    calculate_interval: 1h
    error_budget_size: 0.001
    objectives:
      - expr: alb_p90_response_time <= 1.0
      - expr: component_response_time <= 1.0

dashboard: dashboard.jsonnet
```

#### required_version

 the `required_version` accepts a version constraint string, which specifies which versions of shimesaba can be used with your configuration.

#### metrics

The `metrics` accepts list of Mackerel metrics configure.  
`shimesaba` gets the mackerel metric specified in this list.   
The metrics described in this list can be found in the `definitions` settings described below.
Each setting item in the list is as follows

##### id 

Required.  
An identifier to refer to in `definitions`.
Must be unique in the list

##### name 

Required.  
Metric identifier on Mackerel

##### type 

Required.  
The type of metric. Host metric must set `host` and service metric must set` service`.

##### service_name

Required.  
Specify the name of the service to which the metric belongs

##### roles

Optional, only type=`host`  
Specifies the role when searching for hosts that are subject to host metrics.

##### host_name

Optional, only type=`host`  
Specify the host name when searching for the host that is the target of host metrics.

##### aggregation_interval

Optional, default=1m
It's time to aggregate the metrics.
This is also the unit for determining SLO violations.
For example, if you calculate SLI using a metric with an aggregation interval of 5 minutes, you will get an SLO violation check in 5 minute increments.

##### aggregation_method

Optional, default=max
How to aggregate metrics. There are `max`,` total`, `avg`.

##### interpolated_value

Optional
Specifies the value to interpolate if the data point is missing.
If not specified, it will be treated as nil in the expression.

#### definitions

The `definitions` accepts list of SLI/SLO definition configure.   
6 Mackerel service metrics are posted per definition.  

For example, if id is `latency`, the following service metric will be posted.
- `shimesaba.error_budget.latency`: Current error budget remaining number (unit:minutes)
- `shimesaba.error_budget_percentage.latency`: percentage of current error budget remaining. If it exceeds 100%, the error budget is used up.
- `shimesaba.error_budget_consumption.latency`: Error budget newly consumed in this calculation window (unit:minutes)
- `shimesaba.error_budget_consumption_percentage.latency`: Percentage of newly consumed error budget in this calculation window 
- `shimesaba.failure_time.latency`: Time of SLO violation within the rolling window time frame (unit:minutes)
- `shimesaba.uptime.latency`: Time that can be treated as normal operation within the time frame of the rolling window (unit:minutes)  

Each setting item in the list is as follows  

##### id 

Required.  
The identifier of `definition`. Based on this identifier, the service metric masterpiece at the time of posting is determined.  
Must be unique in the list.

##### service_name

Required.  
The service to which the service metric is posted

##### time_frame

Required. 
The size of the time frame of the rolling window.  
For example, if you specify 40320 minutes, the error budget will be calculated for the SLI for the last 4 weeks.  

##### calculate_interval

Required.  

The shift width of the rolling window. Service metrics are posted to Mackerel at individually specified time intervals.  
This width is recommended to be shorter than 1440 minutes (1 day) because Mackerel ignores postings of time stamp metrics before 24 hours *1.  

*1 [https://mackerel.io/ja/api-docs/entry/service-metrics#post](https://mackerel.io/ja/api-docs/entry/service-metrics#post)

We recommend running shimesaba every hour with `calculate_interval` set to 60 minutes (1 hour).

##### error_budget_size:

Required.  
Setting how much error budget should be taken with respect to the width of the time frame of the rolling window.
For example, if `time_frame` is 40320 and you specify 0.001 (0.1%), the size of the error budget will be 40 minutes.
This means that we will tolerate SLO violations of up to 40 minutes in the last 4 weeks.

##### objectives

Required.  
A list of specific SLO definitions.
This is a list of expr.
`expr` defines a Go syntax comparison expression.
You can use the `id` specified in `metrics` like a variable.
The right-hand side of the comparison must always be a numeric literal.
If multiple expr are defined in the objectives, all must be true.
If any of expr are false, it is a violation of SLO.

For example:   
Assuming that you have obtained the metrics `alb_2xx` and `alb_5xx`, you can write the following comparison formula.

```yaml
- expr: rate(alb_2xx, alb_2xx + alb_5xx) >= 0.95
```

`rate()` is a function prepared to safely execute division while avoiding division by zero.
The meaning of this comparison formula is `If the HTTP request rate is 95% or higher, the service is healthy`.

### Environment variable `SSMWRAP_PATHS`

It incorporates [github.com/handlename/ssmwrap](https://github.com/handlename/ssmwrap) for parameter management.  
If you specify the path of the Parameter Store of AWS Systems Manager separated by commas, it will be output to the environment variable.  
Useful when used as a Lambda function.  

### Usage Dashboard subcommand.

This subcommand can only be used when acting as a CLI.  
If the dashboard of the config file contains the dashboard definition file, you can manage the dashboard JSON using Go Template.

For example, you can build a simple dashboard by defining a json file like the one below.

dashboard.jsonnet
```jsonnet
local errorBudgetCounter(x, y, def_id, title) = {
  type: 'value',
  title: title,
  layout: {
    x: x,
    y: y,
    width: 10,
    height: 5,
  },
  metric: {
    type: 'service',
    name: 'shimesaba.error_budget.' + def_id,
    serviceName: 'shimesaba',
  },
  graph: null,
  range: null,
  fractionSize: 0,
  suffix: 'min',
};
{
  title: 'SLI/SLO',
  urlPath: '4oequPJEwwd',
  memo: '',
  widgets: [
    errorBudgetCounter(0, 0, 'availability', ''),
    errorBudgetCounter(10, 0, 'latency', ''),
    {
      type: 'markdown',
      title: 'SLO Definitions',
      layout: {
        x: 20,
        y: 0,
        width: 5,
        height: 20,
      },
      markdown: '{{file `definitions.md` | json_escape }}',
    },
  ],
}
```

definitions.md
```markdown
{{ range $def_id, $def := .Definitions }}
## SLO {{ $def_id }}

- TimeFrame      : {{ $def.TimeFrame }}
- ErrorBudgetSize: {{ $def.ErrorBudgetSizeDuration }}  


{{ range $def.Objectives }}
- {{ . }}
{{ end }}
{{ end }}
```

## LICENSE

MIT
