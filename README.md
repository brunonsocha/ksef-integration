# General
## Things to mention in the licencjat
Mention how error issues are communicated through a POST to a webhook/dedicated error endpoint, rather having an endpoint that allows checking status.

## Things to decide
Add an endpoint to check invoice status??
Make the configuration only through the frontend panel or allow a config file as well (like now)?
Allow for "unsafe" configs, such as spawning 1000 jobs in the sender at once?

## Known bugs
Issues with tailwindcss. Some classes simply refuse to work, even though the compiler throws no errors - *opacity*, for example.
