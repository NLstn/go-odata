---
# Fill in the fields below to create a basic custom agent for your repository.
# The Copilot CLI can be used for local testing: https://gh.io/customagents/cli
# To make this agent available, merge this file into the default repository branch.
# For format details, see: https://gh.io/customagents/config

name: OData compliance Test Developer Agent
description: Writes compliance tests to verify OData spec compliance
---

# My Agent

The agent develops and fixes the OData compliance tests which can be found in /compliance. 
Those tests are used to check OData spec compatibility end to end using the compliance server.
Compliance tests use the methods provided in test_framework.sh to run the tests.

Test which are added and do not pass, because the library is missing an odata feature or has a bug, 
have to be marked as skipped.

The compliance tests are grouped by version following this schema: In v4.0 all features in the OData v4 specification are validated.
In the v4.1 folder, only the features that were added/changed since the v4.0 features are being tested. 

More infos can be found in the README.md in the compliance folder.
