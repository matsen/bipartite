---
name: bip-board
description: "Manage GitHub project boards. Add, move, and remove issues; boards auto-resolve from repo→channel mappings in sources.yml."
---

# /bip-board

Manage GitHub project boards.
Boards are resolved automatically from repo → channel → board mappings in sources.yml.

Subcommands are `list`, `add`, `move`, `remove`, and `refresh-cache`; `bip board <cmd> --help` has their flags and examples.

## Board Resolution

The board is automatically resolved via channel mappings:
1. Look up repo's channel from `code`/`writing` array in sources.yml
2. Look up channel's board from `boards` mapping

Example sources.yml:
```json
{
  "boards": {
    "dasm2": "matsengrp/30",
    "loris": "matsengrp/29"
  },
  "code": [
    {"repo": "matsengrp/dasm2-experiments", "channel": "dasm2"},
    {"repo": "matsengrp/loris-experiments", "channel": "loris"}
  ]
}
```

`--to owner/number` overrides this resolution for `add`, `move`, and `remove`.
