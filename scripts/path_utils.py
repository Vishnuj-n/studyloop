import os

def validate_path(path, root):
    """
    Validates that a path is within the project root to prevent directory traversal.
    """
    abs_path = os.path.abspath(path)
    abs_root = os.path.abspath(root)
    if os.path.commonpath([abs_path, abs_root]) != abs_root:
        raise ValueError(f"Access denied: {path} escapes project root {root}")
    return abs_path
