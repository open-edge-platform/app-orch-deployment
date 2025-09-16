#!/usr/bin/env python3
# SPDX-FileCopyrightText: 2025 Siemens AG
# SPDX-License-Identifier: Apache-2.0
"""
Merge ConnectRPC individual OpenAPI files into a single consolidated specification.
Converts OpenAPI 3.1.0 format to 3.0.3 for compatibility.
"""

import yaml
from pathlib import Path
import sys
import os

def handle_oneof_schema(oneof_list, data):
    """Handle oneOf schemas conversion from OpenAPI 3.1 to 3.0.3 format."""
    has_null = any(item.get('type') == 'null' for item in oneof_list)
    non_null_items = [item for item in oneof_list if item.get('type') != 'null']
    
    if has_null and len(non_null_items) == 1:
        # Simple case: oneOf with one type + null
        schema = convert_3_1_to_3_0(non_null_items[0])
        schema['nullable'] = True
        return schema
    elif has_null and len(non_null_items) > 1:
        # Complex case: oneOf with multiple types + null (like google.protobuf.Value)
        # For OpenAPI 3.0.3 compatibility, use a generic object schema
        return {
            'type': 'object',
            'nullable': True,
            'additionalProperties': True,
            'description': data.get('description', 'Dynamic value that can be of any type')
        }
    elif len(oneof_list) == 1:
        # Single item oneOf, just use the item directly
        return convert_3_1_to_3_0(oneof_list[0])
    else:
        # Standard oneOf without null
        return convert_3_1_to_3_0(oneof_list)

def convert_examples_field(examples_value):
    """Convert OpenAPI 3.1 'examples' array to 3.0.3 'example' single value."""
    return examples_value[0] if examples_value else None

def convert_openapi_version(version_value):
    """Convert OpenAPI version from 3.1.x to 3.0.3."""
    return '3.0.3' if version_value.startswith('3.1') else version_value

def handle_const_enum_conversion(key, value, has_const, const_value, new_data):
    """Handle const and enum field conversions for OpenAPI 3.0.3 compatibility."""
    if key == 'const':
        return True, value  # Return updated has_const and const_value
    elif key == 'enum' and has_const:
        # If we have both const and enum, prefer const (convert to enum)
        new_data['enum'] = [const_value]
        return has_const, const_value
    else:
        new_data[key] = convert_3_1_to_3_0(value)
        return has_const, const_value

def process_dict_field(key, value, data, new_data, has_const, const_value):
    """Process a single field in a dictionary during OpenAPI conversion."""
    if key == 'examples' and isinstance(value, list):
        new_data['example'] = convert_examples_field(value)
    elif key == 'openapi':
        new_data[key] = convert_openapi_version(value)
    elif key == 'oneOf' and isinstance(value, list):
        # Handle oneOf schemas using extracted function
        result = handle_oneof_schema(value, data)
        if isinstance(result, dict) and 'type' in result:
            # If we got a simplified schema, replace the entire data structure
            return result, has_const, const_value
        else:
            new_data[key] = result
    else:
        has_const, const_value = handle_const_enum_conversion(key, value, has_const, const_value, new_data)
    
    return None, has_const, const_value

def convert_3_1_to_3_0(data):
    """Convert OpenAPI 3.1.0 format to 3.0.3 format."""
    # Handle non-dict types
    if isinstance(data, list):
        return [convert_3_1_to_3_0(item) for item in data]
    elif not isinstance(data, dict):
        return data
    
    # Process dictionary data
    new_data = {}
    has_const = False
    const_value = None
    
    for key, value in data.items():
        # Process each field and check for early return (oneOf simplification)
        early_return, has_const, const_value = process_dict_field(
            key, value, data, new_data, has_const, const_value
        )
        if early_return is not None:
            return early_return
    
    # Handle const conversion after processing all fields
    if has_const and 'enum' not in new_data:
        new_data['enum'] = [const_value]
        
    return new_data

def main():
    """Main function to merge OpenAPI files."""
    if len(sys.argv) < 3:
        print("Usage: merge_openapi.py <input_dir> <output_file> [title] [description] [version]")
        sys.exit(1)
    
    input_dir = Path(sys.argv[1])
    output_file = Path(sys.argv[2])
    title = sys.argv[3] if len(sys.argv) > 3 else "API"
    description = sys.argv[4] if len(sys.argv) > 4 else "API service"
    version = sys.argv[5] if len(sys.argv) > 5 else "1.0.0"
    
    # Find all OpenAPI files with better pattern matching
    openapi_patterns = ['*.openapi.yaml', '*.openapi.yml', '*.swagger.yaml', '*.swagger.yml']
    files = []
    for pattern in openapi_patterns:
        files.extend(input_dir.rglob(pattern))
    
    if not files:
        print(f"No OpenAPI files found in {input_dir}")
        print(f"Looked for patterns: {', '.join(openapi_patterns)}")
        sys.exit(1)
    
    print(f"Found {len(files)} OpenAPI files to merge:")
    for f in sorted(files):  # Sort for consistent output
        print(f"  - {f}")
    
    # Initialize merged spec with more complete structure
    merged = {
        'openapi': '3.0.3',
        'info': {
            'title': title,
            'description': description,
            'version': version
        },
        'paths': {},
        'components': {
            'schemas': {},
            'responses': {},
            'parameters': {},
            'examples': {},
            'requestBodies': {},
            'headers': {},
            'securitySchemes': {},
            'links': {},
            'callbacks': {}
        },
        'servers': [],
        'security': [],
        'tags': []
    }
    
    # Track merged statistics
    stats = {
        'paths': 0,
        'schemas': 0,
        'errors': 0,
        'warnings': 0
    }
    
    # Merge all files
    for f in files:
        try:
            with open(f, 'r', encoding='utf-8') as file:
                content = yaml.safe_load(file)
                
            if not content:
                print(f"Warning: Empty file {f}")
                stats['warnings'] += 1
                continue
                
            # Merge paths
            if 'paths' in content:
                paths_count = len(content['paths'])
                merged['paths'].update(content['paths'])
                stats['paths'] += paths_count
                print(f"  Merged {paths_count} paths from {f.name}")
                
            # Merge components with 3.1 to 3.0 conversion
            if 'components' in content:
                components = convert_3_1_to_3_0(content['components'])
                for component_type in merged['components']:
                    if component_type in components:
                        merged['components'][component_type].update(components[component_type])
                        if component_type == 'schemas':
                            stats['schemas'] += len(components[component_type])
                            
            # Merge servers if present
            if 'servers' in content and isinstance(content['servers'], list):
                for server in content['servers']:
                    if server not in merged['servers']:
                        merged['servers'].append(server)
                        
            # Merge security schemes
            if 'security' in content and isinstance(content['security'], list):
                for security in content['security']:
                    if security not in merged['security']:
                        merged['security'].append(security)
                        
            # Merge tags
            if 'tags' in content and isinstance(content['tags'], list):
                for tag in content['tags']:
                    if tag not in merged['tags']:
                        merged['tags'].append(tag)
                
        except Exception as e:
            print(f"Error processing {f}: {e}")
            stats['errors'] += 1
            if stats['errors'] > len(files) // 2:  # Fail if more than half the files fail
                sys.exit(1)
    
    # Clean up empty components
    merged['components'] = {k: v for k, v in merged['components'].items() if v}
    
    # Clean up empty top-level arrays
    if not merged['servers']:
        del merged['servers']
    if not merged['security']:
        del merged['security']
    if not merged['tags']:
        del merged['tags']
    
    # Ensure output directory exists
    output_file.parent.mkdir(parents=True, exist_ok=True)
    
    # Write merged spec with better formatting
    try:
        with open(output_file, 'w', encoding='utf-8') as file:
            yaml.dump(merged, file, default_flow_style=False, sort_keys=False, 
                     allow_unicode=True, width=120, indent=2)
        print(f"\nSuccessfully merged OpenAPI spec to {output_file}")
        print(f"  - Total paths: {stats['paths']}")
        print(f"  - Total schemas: {stats['schemas']}")
        print(f"  - Components: {list(merged['components'].keys())}")
        if stats['warnings']:
            print(f"  - Warnings: {stats['warnings']}")
        if stats['errors']:
            print(f"  - Errors: {stats['errors']}")
    except Exception as e:
        print(f"Error writing output file {output_file}: {e}")
        sys.exit(1)

if __name__ == '__main__':
    main()
