# Time window catcher

Test how it works time series in golang

It creates a time series window where it stores the OK and KO responses of a polling server.

This should be a test of how analytics sites checks and creates alerts using a threshold based on some metrics.

What I want to achieve is:

- create alerts based on http response
- create alerts based on system performance (cpu or memory)