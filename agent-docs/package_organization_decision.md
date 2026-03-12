# Package Organization Decision

## Why Same-Package Instead of Separate Packages?

The original plan was to extract focus/, agents/, and animation/ as separate packages within internal/ui/. However, during implementation, this approach caused circular dependency issues.

### The Problem

Attempting to create separate packages like:
- internal/ui/focus/manager.go with FocusManager struct
- internal/ui/agents/manager.go with AgentManager struct
- internal/ui/animation/handlers.go with animation handlers

Resulted in:
- Import cycles: ui imports focus/agents/animation, but these packages need access to ui.UIModel and ui.SidebarModel
- Complex dependency graph requiring interfaces for everything

### The Solution

Chose to keep the extracted code in the ui package with file-based organization:
- internal/ui/focus.go - FocusManager and focus navigation
- internal/ui/agents.go - Agent management functions and handlers
- internal/ui/animation_handlers.go - Animation tick handlers
- internal/ui/model_accessors.go - UIModel accessor methods (test helpers)

### Benefits

1. No circular dependencies: All functions share the same package context
2. Simpler code organization: Clear file separation without complex module boundaries
3. Easier refactoring: Functions can be moved between files without breaking imports
4. Better testability: Tests have direct access to all internal types
5. Maintains separation of concerns: Each file has a clear, focused responsibility

### Trade-offs

- Slightly larger ui package (still manageable at ~1800 lines)
- Functions are not independently importable by other packages (not needed anyway)
- Requires careful naming to avoid collisions (handled with clear prefixes)

This approach achieves the code organization goals without the complexity overhead of package boundaries.
