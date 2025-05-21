# Telemetry and data collection

The Contrast CLI sends telemetry data to Edgeless Systems when you use CLI commands.
This allows to understand how Contrast is used and to improve it.

The CLI sends the following data:

* The CLI version
* The CLI target OS and architecture (GOOS and GOARCH)
* The command that was run
* The kind of error that occurred (if any)

The CLI *doesn't* collect sensitive information.
The implementation is open-source and can be reviewed.

IP addresses may be processed or stored for security purposes.

The data that the CLI collects adheres to the Edgeless Systems [privacy policy](https://www.edgeless.systems/privacy).

You can disable telemetry by setting the environment variable `DO_NOT_TRACK=1` before running the CLI.
