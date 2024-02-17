# Pipe

Pipe Widget is used for complex logic orchestration, serving as an alternative to Flow.

**Warning**:

Pipe Widget is an experimental feature and not recommended for production use.

**Usage Scenario**

Generating DSL from a graphical interface, implementing simple functional logic extensions on the application side.

## DSL

CLI: https://github.com/YaoApp/yao-dev-app/blob/main/pipes/cli/translator.pip.yao

WEB: https://github.com/YaoApp/yao-dev-app/blob/main/pipes/web/translator.pip.yao

## Node Types

| Type        | options                      | Description                                               |
| ----------- | ---------------------------- | --------------------------------------------------------- |
| Yao Process | `name`, `args`               | run yao process                                           |
| Switch      |                              | conditional branch                                        |
| AI          | `prompts`, `model`, `option` | AI interface                                              |
| Request     |                              | HTTP request (not supported yet, use yao process instead) |
| User Input  | `ui` (cli/web/...)           | user input interface                                      |

for more details, refer to the DSL demo.

## Process

Refer to unit test programs for examples.

### pipes.<Widget.ID>

Run Pipe

```bash
yao run pipes.<Widget.ID> [args...]
```

If interrupted by user input interface, it returns a context ID for resuming execution.

### pipe.Run

Run Pipe, equivalent to `pipes.<Widget.ID>`

```bash
yao run pipe.Run <Widget.ID> [args...]
```

### pipe.Create

Pass DSL text to create and run Pipe

```bash
yao run pipe.Create <DSL> [args...]
```

### pipe.CreateWith

Pass DSL text to create and run Pipe

```bash
yao run pipe.CreateWith <DSL> '::{"foo":"bar"}' [args...]
```

### pipe.Resume

Resume execution, used for context restoration

```bash
yao run pipe.Resume <Context.ID> [args...]
```

### pipe.ResumeWith

Resume execution, used for context restoration

```bash
yao run pipe.ResumeWith <Context.ID> '::{"foo":"bar"}' [args...]
```

### pipe.Close

Close Pipe

```bash
yao run pipe.Close <Context.ID>
```

## Features

- [x] **Yao Process Node** Support for running yao process
- [x] **Switch Node** Conditional branch
- [x] **AI Node** AI interface
- [x] **User Input Node** User input interface
- [ ] **Request Node** Support for Http Request
- [ ] **Hooks** Progress report for hook integration
