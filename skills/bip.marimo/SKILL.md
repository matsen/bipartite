---
name: bip.marimo
description: Generate and edit marimo reactive notebooks correctly. Auto-triggers when working with marimo .py notebook files.
---

# Marimo Notebooks

Reactive Python notebooks stored as plain `.py` files.

**Docs**: https://docs.marimo.io/
**API Reference**: https://docs.marimo.io/api/

## Environment

Use **pixi** for all marimo commands:

```bash
pixi run marimo edit notebook.py
pixi run marimo run notebook.py
pixi run marimo check notebook.py
pixi run marimo export html notebook.py -o notebook.html
pixi run marimo export ipynb notebook.py -o notebook.ipynb
pixi run marimo convert notebook.ipynb -o notebook.py
```

**Never** use `--sandbox` flag (creates isolated env, ignores your packages).

## Cell Structure (Critical)

Every marimo notebook starts with:
```python
import marimo
app = marimo.App()
```

Each cell is decorated with `@app.cell`:
```python
@app.cell
def _():
    import marimo as mo
    import pandas as pd
    return mo, pd

@app.cell
def _(mo):
    slider = mo.ui.slider(0, 100, value=50, label="Amount")
    slider
    return (slider,)

@app.cell
def _(mo, slider):
    # Auto-runs when slider changes
    mo.md(f"Value: **{slider.value}**")
    return
```

### Rules

1. **Return variables** other cells need via `return (var1, var2,)` tuple
2. **Declare dependencies** as function parameters: `def _(mo, slider):` means this cell uses `mo` and `slider`
3. **No circular dependencies** — `marimo check` catches these
4. **One definition per variable** — each variable defined in exactly one cell
5. **Display by putting expression last** in cell, or use `mo.output.replace()`
6. After editing, `pixi run marimo check` validates notebook structure (the PostToolUse hook runs this automatically)

## Common APIs

```python
import marimo as mo

# Markdown with interpolation
mo.md(f"# Title\nValue is **{x}**")

# UI elements (reactive)
slider = mo.ui.slider(0, 100, value=50, label="Amount")
dropdown = mo.ui.dropdown(["a", "b", "c"], label="Choice")
text = mo.ui.text(placeholder="Enter name")
checkbox = mo.ui.checkbox(label="Enable")
button = mo.ui.button(label="Click me")

# Access values
slider.value  # returns current value

# Layout
mo.hstack([elem1, elem2])  # horizontal
mo.vstack([elem1, elem2])  # vertical
mo.accordion({"Section": content})
mo.tabs({"Tab1": content1, "Tab2": content2})

# Output / control flow
mo.output.replace(content)  # replace cell output
mo.stop(condition, mo.md("Stopped"))  # conditional halt
```
