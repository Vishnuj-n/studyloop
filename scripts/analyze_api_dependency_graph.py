#!/usr/bin/env python3
import os
import re
import sys
import argparse

# Path definitions
DEFAULT_API_JS = "frontend/src/services/appApi.js"
DEFAULT_FRONTEND_DIR = "frontend/src"
DEFAULT_GO_DIR = "internal"

# Exclude patterns
EXCLUDE_GO_TEST = re.compile(r"_test\.go$")
EXCLUDE_FRONTEND_TEST = re.compile(r"\.(test|spec)\.(js|ts)$")

# Regex compilation
JS_EXPORT_FUNC = re.compile(r"export\s+(?:async\s+)?function\s+([a-zA-Z0-9_]+)\s*\(")
WAILS_BRIDGE_CALL = re.compile(r"(?:appBridge\(\)|bridge)\.([a-zA-Z0-9_]+)\(")

# Go receiver method: func (a *App) Name(...)
GO_APP_METHOD = re.compile(r"func\s*\(\s*\w+\s+\*?App\s*\)\s*([a-zA-Z0-9_]+)\s*\(")
# Go general method: func (r *Repository) Name(...)
GO_METHOD = re.compile(r"func\s*\(\s*\w+\s+\*?([a-zA-Z0-9_]+)\s*\)\s*([a-zA-Z0-9_]+)\s*\(")
# Go general function: func Name(...)
GO_FUNC = re.compile(r"func\s+([a-zA-Z0-9_]+)\s*\(")

def parse_app_api(filepath):
    """
    Parses appApi.js and returns a mapping:
    js_func_name -> wails_backend_method_name
    """
    exports = {}
    current_export = None
    
    if not os.path.exists(filepath):
        print(f"Error: API file not found at {filepath}", file=sys.stderr)
        return exports

    with open(filepath, "r", encoding="utf-8", errors="ignore") as f:
        content = f.read()
        
    # We split by lines to process block scopes simply, or use a regex-based parser
    lines = content.splitlines()
    for line in lines:
        exp_match = JS_EXPORT_FUNC.search(line)
        if exp_match:
            current_export = exp_match.group(1)
            exports[current_export] = None
        
        if current_export:
            bridge_match = WAILS_BRIDGE_CALL.search(line)
            if bridge_match:
                exports[current_export] = bridge_match.group(1)
                current_export = None
                
    return exports

def find_js_usages(frontend_dir, api_js_path, js_functions):
    """
    Scans the frontend directory to see which exported JS functions are actually called.
    """
    usages = {func: 0 for func in js_functions}
    
    for root, _, files in os.walk(frontend_dir):
        for file in files:
            filepath = os.path.join(root, file)
            # Skip the api file itself and tests
            if os.path.abspath(filepath) == os.path.abspath(api_js_path):
                continue
            if EXCLUDE_FRONTEND_TEST.search(file):
                continue
                
            if file.endswith((".js", ".ts", ".vue")):
                try:
                    with open(filepath, "r", encoding="utf-8", errors="ignore") as f:
                        content = f.read()
                    for func in js_functions:
                        # Search for word boundary usage of the function
                        if re.search(r"\b" + re.escape(func) + r"\b", content):
                            usages[func] += 1
                except Exception as e:
                    print(f"Error reading frontend file {filepath}: {e}", file=sys.stderr)
                    
    return usages

def parse_go_definitions(go_dir):
    """
    Scans Go codebase to collect:
    1. App methods (bound to Wails)
    2. Other Go methods / functions
    Returns:
      app_methods: set of method names
      other_funcs: dict of name -> { 'type': 'function'/'method', 'receiver': receiver_type, 'file': file }
    """
    app_methods = set()
    other_funcs = {}
    
    for root, _, files in os.walk(go_dir):
        for file in files:
            if EXCLUDE_GO_TEST.search(file) or not file.endswith(".go"):
                continue
            filepath = os.path.join(root, file)
            rel_path = os.path.relpath(filepath, go_dir)
            
            try:
                with open(filepath, "r", encoding="utf-8", errors="ignore") as f:
                    for line_num, line in enumerate(f, 1):
                        # 1. Check App methods
                        app_match = GO_APP_METHOD.search(line)
                        if app_match:
                            app_methods.add(app_match.group(1))
                            continue
                            
                        # 2. Check other struct methods
                        method_match = GO_METHOD.search(line)
                        if method_match:
                            recv = method_match.group(1)
                            name = method_match.group(2)
                            other_funcs[name] = {
                                "type": "method",
                                "receiver": recv,
                                "file": f"{rel_path}:{line_num}"
                            }
                            continue
                            
                        # 3. Check plain functions
                        func_match = GO_FUNC.search(line)
                        if func_match:
                            name = func_match.group(1)
                            # Ignore main or common reserved words
                            if name in ("main", "init"):
                                continue
                            other_funcs[name] = {
                                "type": "function",
                                "receiver": None,
                                "file": f"{rel_path}:{line_num}"
                            }
            except Exception as e:
                print(f"Error reading Go file {filepath}: {e}", file=sys.stderr)
                
    return app_methods, other_funcs

def build_go_call_graph(go_dir, all_go_funcs):
    """
    Builds an adjacency list representation of Go function calls:
    caller -> set(callees)
    """
    call_graph = {}
    func_names = set(all_go_funcs.keys())
    
    # We first map which functions/methods are defined in which files
    file_to_funcs = {}
    for func_name, info in all_go_funcs.items():
        filename = info["file"].split(":")[0]
        file_to_funcs.setdefault(filename, []).append(func_name)
        
    for root, _, files in os.walk(go_dir):
        for file in files:
            if EXCLUDE_GO_TEST.search(file) or not file.endswith(".go"):
                continue
            filepath = os.path.join(root, file)
            rel_path = os.path.relpath(filepath, go_dir)
            
            try:
                with open(filepath, "r", encoding="utf-8", errors="ignore") as f:
                    content = f.read()
                
                # Simple approximation: if a function name is referenced in the file,
                # any caller defined in this file might call it.
                # To make this better, we'll map references by scanning.
                for target_func in func_names:
                    # Don't count the definition itself as a call
                    def_file = all_go_funcs[target_func]["file"].split(":")[0]
                    
                    # Look for references of target_func in content
                    matches = list(re.finditer(r"\b" + re.escape(target_func) + r"\b", content))
                    
                    # We subtract 1 match if this file contains the definition
                    expected_matches = 1 if rel_path == def_file else 0
                    if len(matches) > expected_matches:
                        # File references target_func. Let's find who the callers are.
                        # For simplicity, we attribute references in this file to the Go methods/functions
                        # defined within this file.
                        callers = file_to_funcs.get(rel_path, [])
                        for caller in callers:
                            if caller != target_func:
                                call_graph.setdefault(caller, set()).add(target_func)
            except Exception as e:
                pass
                
    return call_graph

def main():
    parser = argparse.ArgumentParser(description="Analyze API and backend call graphs to find redundant code.")
    parser.add_argument("--api", default=DEFAULT_API_JS, help="Path to appApi.js")
    parser.add_argument("--frontend", default=DEFAULT_FRONTEND_DIR, help="Path to frontend source directory")
    parser.add_argument("--go", default=DEFAULT_GO_DIR, help="Path to Go internal directory")
    parser.add_argument("--output", help="Output path for the report (Markdown)")
    
    args = parser.parse_args()
    
    script_dir = os.path.dirname(os.path.abspath(__file__))
    project_root = os.path.dirname(script_dir)
    
    api_js_path = os.path.join(project_root, args.api)
    frontend_dir = os.path.join(project_root, args.frontend)
    go_dir = os.path.join(project_root, args.go)
    
    print("Step 1: Parsing appApi.js...")
    api_mappings = parse_app_api(api_js_path)
    
    print("Step 2: Checking JS usages in frontend...")
    js_usages = find_js_usages(frontend_dir, api_js_path, api_mappings.keys())
    
    print("Step 3: Parsing Go definitions...")
    app_methods, other_go_funcs = parse_go_definitions(go_dir)
    
    # Classify App receiver methods:
    # - Wails Lifecycle: Startup, Shutdown, DomReady (exported, but system-called)
    # - Wails API Endpoints: Exported (starts with uppercase)
    # - App Helpers: Unexported (starts with lowercase)
    wails_lifecycle_methods = set()
    wails_api_methods = set()
    app_helper_methods = set()
    
    for m in app_methods:
        if m in ("Startup", "Shutdown", "DomReady"):
            wails_lifecycle_methods.add(m)
        elif m[0].isupper():
            wails_api_methods.add(m)
        else:
            app_helper_methods.add(m)
            
    # Merge all functions for reachability tracing
    # We treat AppHelpers as internal methods of App
    all_go_funcs = {}
    for m in wails_lifecycle_methods:
        all_go_funcs[m] = {"type": "WailsLifecycle", "receiver": "App", "file": "internal/app/app.go"}
    for m in wails_api_methods:
        all_go_funcs[m] = {"type": "WailsAPI", "receiver": "App", "file": "internal/app/app.go"}
    for m in app_helper_methods:
        all_go_funcs[m] = {"type": "AppHelper", "receiver": "App", "file": "internal/app/app.go"}
        
    all_go_funcs.update(other_go_funcs)
    
    print("Step 4: Building Go Call Graph...")
    call_graph = build_go_call_graph(go_dir, all_go_funcs)
    
    # Trace reachability starting from:
    # 1. Active frontend-invoked Wails APIs
    # 2. Wails lifecycle entry points (Startup, Shutdown, DomReady)
    active_js_funcs = [js_f for js_f, count in js_usages.items() if count > 0]
    active_app_methods = set()
    for js_f in active_js_funcs:
        go_m = api_mappings.get(js_f)
        if go_m and go_m in wails_api_methods:
            active_app_methods.add(go_m)
            
    # Start tracing from active frontend APIs + lifecycle methods
    entry_points = active_app_methods.union(wails_lifecycle_methods)
    reachable = set(entry_points)
    queue = list(entry_points)
    
    while queue:
        current = queue.pop(0)
        callees = call_graph.get(current, set())
        for callee in callees:
            if callee not in reachable:
                reachable.add(callee)
                queue.append(callee)
                
    # Prepare Report
    lines = []
    lines.append("# API & Function Dependency Report")
    lines.append("This report lists unused API endpoints, dead Go backend code, and reachability stats.\n")
    
    # Section 1: Frontend appApi.js usage
    lines.append("## 1. Frontend API (`appApi.js`) Usages")
    lines.append("Lists functions defined in `appApi.js` and their usage count in the frontend.")
    lines.append("| JS Function | Calls Wails Method | Usage Count | Status |")
    lines.append("| --- | --- | --- | --- |")
    
    unused_js = []
    for js_f, wails_m in sorted(api_mappings.items()):
        count = js_usages.get(js_f, 0)
        status = "✅ Active" if count > 0 else "❌ Unused"
        if count == 0:
            unused_js.append(js_f)
        lines.append(f"| `{js_f}` | `{wails_m}` | {count} | {status} |")
    lines.append("")
    
    # Section 2: Wails Go App Methods (Exported Only)
    lines.append("## 2. Go Wails `App` API Endpoints")
    lines.append("Lists exported API endpoints defined on `App` in the backend and whether they are invoked by `appApi.js` (excluding standard Wails lifecycle methods).")
    lines.append("| Wails Method | Reachable from Frontend |")
    lines.append("| --- | --- |")
    
    for m in sorted(wails_api_methods):
        status = "✅ Yes" if m in active_app_methods else "❌ No (Unused API)"
        lines.append(f"| `{m}` | {status} |")
    lines.append("")
    
    # Section 3: Go Reachability & Dead Code
    lines.append("## 3. Go Internal Reachability Analysis")
    lines.append("Lists internal Go functions/methods and unexported `App` helpers that are **not reachable** from any active frontend API or lifecycle entry point (excluding tests).")
    lines.append("| Function/Method | File | Receiver | Type |")
    lines.append("| --- | --- | --- | --- |")
    
    unreachable_count = 0
    for name, info in sorted(all_go_funcs.items()):
        # Wails API endpoints and Wails lifecycle methods are reported in Section 2, or are top-level entry points.
        if info["type"] in ("WailsAPI", "WailsLifecycle"):
            continue
        if name not in reachable:
            unreachable_count += 1
            recv = f"`{info['receiver']}`" if info["receiver"] else "None"
            lines.append(f"| `{name}` | `{info['file']}` | {recv} | `{info['type']}` |")
            
    if unreachable_count == 0:
        lines.append("| None | - | - | - |")
    lines.append("")
    
    lines.append("## 4. Mermaid Flowchart (Active Core API)")
    lines.append("```mermaid")
    lines.append("flowchart TD")
    # Only render top level relationships to avoid massive chart
    lines.append("    subgraph Frontend")
    for js_f in sorted(active_js_funcs)[:15]: # Limit to top 15 for readability
        lines.append(f"        F_{js_f}[{js_f}]")
    lines.append("    end")
    lines.append("    subgraph Go_Wails_Bridge")
    for m in sorted(active_app_methods)[:15]:
        lines.append(f"        G_{m}[App.{m}]")
    lines.append("    end")
    
    # Connect frontend to backend
    for js_f in sorted(active_js_funcs)[:15]:
        wails_m = api_mappings.get(js_f)
        if wails_m in active_app_methods:
            lines.append(f"        F_{js_f} --> G_{wails_m}")
            
    lines.append("```")
    lines.append("\n*Note: The Mermaid flowchart is capped to the first 15 active endpoints for visual clarity.*")
    
    report = "\n".join(lines)
    
    if args.output:
        out_path = os.path.abspath(args.output)
        with open(out_path, "w", encoding="utf-8") as out_f:
            out_f.write(report)
        print(f"Report written to {out_path}")
    else:
        print(report)

if __name__ == "__main__":
    main()
