# Monitor

This project is a monitoring program to monitor whether the light client synchronization is normal,

whether the transaction is cross-chain, and the user balance

# Configuration

See `config.example` for an example configuration.

## Options

```shell
{
  "lightnode": "0x12345...",                              // the lightnode to sync header
  "waterLine": "5000000000000000000",                     // If the user balance is lower than, an alarm will be triggered, unit : wei
  "changeInterval": "3000",                               // How long does the lightnode height remain unchanged, triggering the alarm, use for near unit : seconds
  "checkHeightCount": "20",                               // How long does the lightnode height not change remain unchanged, triggering the alarm, default 15
}
```

## Env

```shell 
export hooks="https://hooks.slack.com/services/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" // Slack alarm hook, Apply See This https://api.slack.com/messaging/webhooks
```