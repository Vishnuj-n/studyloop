#!/usr/bin/env python3
import os
import re
import sys
import argparse

# Regular expressions for extraction
TEMPLATE_RE = re.compile(r'<template[^>]*>(.*)</template>', re.DOTALL)
SCRIPT_RE = re.compile(r'<script[^>]*>(.*?)</script>', re.DOTALL)
STYLE_RE = re.compile(r'<style([^>]*)>(.*?)</style>', re.DOTALL)

# CSS parser helpers
SELECTOR_CLEAN_RE = re.compile(r'::?[a-zA-Z0-9_-]+(?:\(.*?\))?')  # Remove pseudo-classes/elements like :hover, :not(...)

def extract_blocks(file_content):
    """Extracts template, script, and style blocks from Vue SFC content."""
    template_match = TEMPLATE_RE.search(file_content)
    template_content = template_match.group(1) if template_match else ""

    script_content = ""
    for m in SCRIPT_RE.finditer(file_content):
        script_content += m.group(1) + "\n"

    style_blocks = []
    for m in STYLE_RE.finditer(file_content):
        attrs = m.group(1)
        style_content = m.group(2)
        is_scoped = "scoped" in attrs
        style_blocks.append({
            "attrs": attrs,
            "content": style_content,
            "scoped": is_scoped,
            "start_pos": m.start(2)
        })

    return template_content, script_content, style_blocks

def get_line_number(content, char_index):
    """Calculates the 1-based line number for a character index in content."""
    return content.count('\n', 0, char_index) + 1

def parse_css_rules(style_content, start_pos, full_content):
    """Parses CSS rules and extracts selectors with their line numbers, handling media queries & comments."""
    # Remove CSS comments but keep length and newlines aligned (replace with spaces/newlines)
    cleaned_style = ""
    in_comment = False
    i = 0
    n = len(style_content)
    while i < n:
        if in_comment:
            if i + 1 < n and style_content[i] == '*' and style_content[i+1] == '/':
                cleaned_style += '  '
                in_comment = False
                i += 2
            else:
                char = style_content[i]
                cleaned_style += '\n' if char == '\n' else ' '
                i += 1
        else:
            if i + 1 < n and style_content[i] == '/' and style_content[i+1] == '*':
                cleaned_style += '  '
                in_comment = True
                i += 2
            else:
                cleaned_style += style_content[i]
                i += 1

    rules = []
    stack = []  # Holds (start_index, type)
    current_selector = []
    i = 0
    n = len(cleaned_style)
    
    while i < n:
        char = cleaned_style[i]
        if char == '{':
            selector_text = "".join(current_selector).strip()
            if selector_text.startswith('@keyframes') or selector_text.startswith('@-webkit-keyframes'):
                stack.append((i, "keyframes"))
            elif selector_text.startswith('@media'):
                stack.append((i, "media"))
            else:
                is_in_keyframes = any(t == "keyframes" for _, t in stack)
                if not is_in_keyframes and selector_text:
                    selector_start_idx = i - len(current_selector)
                    rules.append({
                        "selector": selector_text,
                        "line": get_line_number(full_content, start_pos + selector_start_idx)
                    })
                stack.append((i, "selector"))
            current_selector = []
        elif char == '}':
            if stack:
                stack.pop()
            current_selector = []
        else:
            is_parsing_selector = False
            if not stack:
                is_parsing_selector = True
            elif stack[-1][1] == "media":
                is_parsing_selector = True
                
            if is_parsing_selector:
                current_selector.append(char)
                
        i += 1
        
    return rules

def analyze_selector(selector):
    """
    Parses a single selector string (e.g. '.notebook-card h3') 
    and returns its component parts: classes, ids, and tags.
    """
    if any(d in selector for d in [":deep", "::v-deep", "/deep/", ":global"]):
        return [], [], [], True
        
    cleaned = SELECTOR_CLEAN_RE.sub(' ', selector)
    parts = re.split(r'[\s>+~]+', cleaned)
    
    classes = []
    ids = []
    tags = []
    
    for part in parts:
        if not part:
            continue
            
        for cls in re.findall(r'\.([a-zA-Z0-9_-]+)', part):
            classes.append(cls)
            
        for ident in re.findall(r'#([a-zA-Z0-9_-]+)', part):
            ids.append(ident)
            
        tag_match = re.match(r'^([a-zA-Z0-9-]+)', part)
        if tag_match:
            tags.append(tag_match.group(1))
            
    return classes, ids, tags, False

def check_redundant_css(file_path):
    """Analyzes a single Vue file for redundant CSS."""
    try:
        with open(file_path, 'r', encoding='utf-8') as f:
            content = f.read()
    except Exception as e:
        print(f"Error reading {file_path}: {e}", file=sys.stderr)
        return None

    template_content, script_content, style_blocks = extract_blocks(content)
    
    if not style_blocks:
        return []

    template_tokens = set(re.findall(r'[a-zA-Z0-9_-]+', template_content))
    script_tokens = set(re.findall(r'[a-zA-Z0-9_-]+', script_content))
    all_tokens = template_tokens.union(script_tokens)

    # Extract transition names to prevent flagging transition classes as unused
    transition_names = re.findall(r'<[Tt]ransition[^>]*\s:?name=["\']([^"\']+)["\']', template_content)
    for t_name in transition_names:
        for suffix in ["-enter-active", "-leave-active", "-enter-from", "-leave-to", "-enter-to", "-leave-from"]:
            all_tokens.add(t_name + suffix)

    tag_tokens = set(re.findall(r'<([a-zA-Z0-9-]+)', template_content))

    redundant_selectors = []

    for block in style_blocks:
        if not block["scoped"]:
            continue
            
        rules = parse_css_rules(block["content"], block["start_pos"], content)
        
        for rule in rules:
            selector_text = rule["selector"]
            if not selector_text:
                continue
                
            sub_selectors = [s.strip() for s in selector_text.split(',') if s.strip()]
            
            unused_sub_selectors = []
            for sub in sub_selectors:
                classes, ids, tags, ignored = analyze_selector(sub)
                
                if ignored:
                    continue
                
                is_used = False
                
                if classes:
                    if any(cls in all_tokens for cls in classes):
                        is_used = True
                
                if ids:
                    if any(ident in all_tokens for ident in ids):
                        is_used = True
                        
                if not classes and not ids and tags:
                    if all(tag in tag_tokens for tag in tags):
                        is_used = True
                
                if not classes and not ids and not tags:
                    is_used = True
                    
                if not is_used:
                    unused_sub_selectors.append(sub)
            
            if unused_sub_selectors:
                redundant_selectors.append({
                    "line": rule["line"],
                    "selectors": unused_sub_selectors,
                    "original": selector_text
                })

    return redundant_selectors

def main():
    parser = argparse.ArgumentParser(description="Find redundant CSS selectors in Vue SFC files.")
    parser.add_argument("path", nargs="?", default="frontend/src", help="Path to a Vue file or directory containing Vue files.")
    
    args = parser.parse_args()
    
    target_path = args.path
    if not os.path.exists(target_path):
        print(f"Path does not exist: {target_path}", file=sys.stderr)
        sys.exit(1)
        
    vue_files = []
    if os.path.isfile(target_path):
        if target_path.endswith('.vue'):
            vue_files.append(target_path)
    else:
        for root, _, files in os.walk(target_path):
            for file in files:
                if file.endswith('.vue'):
                    vue_files.append(os.path.join(root, file))
                    
    if not vue_files:
        print("No .vue files found.")
        sys.exit(0)
        
    total_redundant = 0
    files_checked = 0
    
    print(f"Scanning {len(vue_files)} Vue file(s) for redundant CSS selectors...")
    print("-" * 80)
    
    for file in sorted(vue_files):
        redundant = check_redundant_css(file)
        if redundant:
            files_checked += 1
            print(f"\n[{file}]")
            for item in redundant:
                total_redundant += len(item["selectors"])
                print(f"  Line {item['line']}: Unused selector(s): {', '.join(item['selectors'])}")
                
    print("-" * 80)
    print(f"Scan complete. Found {total_redundant} redundant CSS selector(s) across {files_checked} file(s).")

if __name__ == "__main__":
    main()
