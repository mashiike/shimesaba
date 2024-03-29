![Latest GitHub release](https://img.shields.io/github/release/mashiike/shimesaba.svg)
![Github Actions test](https://github.com/mashiike/shimesaba/workflows/Test/badge.svg?branch=main)
[![Go Report Card](https://goreportcard.com/badge/mashiike/shimesaba)](https://goreportcard.com/report/mashiike/shimesaba) [![License](https://img.shields.io/badge/license-MIT-blue.svg)](https://github.com/mashiike/shimesaba/blob/master/LICENSE)
# shimesaba

For SRE to operate and monitor services using Mackerel.
## Description

shimesaba is a tool for tracking SLO/ErrorBudget using Mackerel as an SLI measurement service.

- shimesaba evaluates window-based SLOs with monitoring data on Mackerel.
- Post the calculated values (error budget, failure time for SLO violation, uptime etc) by evaluating SLOs . as Mackerel service metric.


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
$ shimesaba -config config.yaml -mackerel-apikey <Mackerel API Key> 
```

```console
NAME:
   shimesaba - A commandline tool for tracking SLO/ErrorBudget using Mackerel as an SLI measurement service.

USAGE:
   shimesaba -config <config file> [command options]

VERSION:
   v1.0.0

COMMANDS:
   run        run shimesaba. this is main feature (deprecated), use no subcommand
   help, h    Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --backfill value                   generate report before n point (default: 3) [$BACKFILL, $SHIMESABA_BACKFILL]
   --config value, -c value           config file path, can set multiple [$CONFIG, $SHIMESABA_CONFIG]
   --debug                            output debug log (default: false) [$SHIMESABA_DEBUG]
   --dry-run                          report output stdout and not put mackerel (default: false) [$SHIMESABA_DRY_RUN]
   --mackerel-apikey value, -k value  for access mackerel API (default: *********) [$MACKEREL_APIKEY, $SHIMESABA_MACKEREL_APIKEY]
   --help, -h                         show help (default: false)
   --version, -v                      print the version (default: false)
```

### as AWS Lambda function

`shimesaba` binary also runs as AWS Lambda function. 
shimesaba implicitly behaves as a run command when run as a bootstrap with a Lambda Function

CLI options can be specified from environment variables. For example, when `MACKEREL_APIKEY` environment variable is set, the value is set to `-mackerel-apikey` option.

Example Lambda functions configuration with [github.com/fujiwara/lambroll](https://github.com/fujiwara/lambroll)

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

The following are the settings for the latest v0.7.0.

YAML format.

```yaml
required_version: ">=1.0.0" # which specifies which versions of shimesaba can be used with your configuration.

# This is a common setting item for error budget calculation.
# It is possible to override the same settings in each SLO definition.
destination:
    service_name: prod          # - The name of the service to which you want to submit the service metric for error budgeting.
    metric_prefix: api          # - Specifies the service metric prefix for error budgeting.
rolling_period: 28d             # - Specify the size of the rolling window to calculate the error budget.
calculate_interval: 1h      # - Settings related to the interval for calculating the error budget.
error_budget_size: 0.1%     # - This setting is related to the size of the error budget.
                            #   If % is used, it is a ratio to the size of the rolling window.
                            #   It is also possible to specify a time such as 1h or 40m.

# Describes the settings for each SLO. SLOs are treated as monitoring rules.
# The definition of each SLO is determined by ORing the monitoring rules that match the conditions specified in `objectives`.
# That is, based on the alerts corresponding to the monitoring rules that match the conditions, the existence of any of the alerts will be judged as SLO violation.
slo:
  # In the availability SLO, if an alert occurs for a monitoring rule name that starts with "SLO availability" 
  #  or an external monitoring rule that ends with "api.example.com", it is considered an SLO violation. 
  - id: availability
    alert_based_sli: # This setting uses Mackerel alerts as SLI.
      - monitor_name_prefix: "SLO availability"
      - monitor_name_suffix: "api.example.com"
        monitor_type: "external" 
  # In the latency SLO, we consider it an SLO violation if an alert occurs for a host metric monitoring rule with a name starting with "SLO availability".
  - id: latency
    error_budget_size: 200m
    alert_based_sli:
      - monitor_name_prefix: "SLO latency"
      - monitor_type: "host"
        try_reassessment: true # This setting attempts to reevaluate an alert using the actual metric only if the type of monitor from which the alert originated is service or host.
```

`slo` takes a list of constituent SLI/SLO definitions.  
6 Mackerel service metrics will be listed per definition. 

For example, if id is `latency` in the above configuration, the following service metric will be posted.
- `api.error_budget.latency`: Current error budget remaining number (unit:minutes)
- `api.error_budget_percentage.latency`: percentage of current error budget remaining. If it exceeds 100%, the error budget is used up.
- `api.error_budget_consumption.latency`: Error budget newly consumed in this calculation window (unit:minutes)
- `api.error_budget_consumption_percentage.latency`: Percentage of newly consumed error budget in this calculation window 
- `api.failure_time.latency`: Time of SLO violation within the rolling window time frame (unit:minutes)
- `api.uptime.latency`: Time that can be treated as normal operation within the time frame of the rolling window (unit:minutes)  

### Manual correction feature

If you enter `downtime:3m` or similar in the reason for closing an alert, the alert will be calculated as if the SLO had been violated for 3 minutes from the time it was opened.

The description "3m" can be any time like `1h`, `40m`, `1h50m`, etc. as well as other settings.
When combined with other statements, half-width spaces are required before and after the above keywords.

### Environment variable `SSMWRAP_PATHS`, `SSMWRAP_NAMES`

It incorporates [github.com/handlename/ssmwrap](https://github.com/handlename/ssmwrap) for parameter management.  
If you specify the path of the Parameter Store of AWS Systems Manager separated by commas, it will be output to the environment variable.  
Useful when used as a Lambda function. 

For example, if you have a secrets named `prod/MACKEREL_APIKEY` in your secrets manager, it is useful to set the following environment variable.

`SSMWRAP_NAMES=/aws/reference/secretsmanager/prod/MACKEREL_APIKEY`

## LICENSE

MIT
