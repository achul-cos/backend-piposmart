import json
import re

jsonl_file = 'C:\\Users\\User\\Documents\\crm_piposmart\\backend\\old_export_snippet.jsonl'
target_file = 'C:\\Users\\User\\Documents\\crm_piposmart\\backend\\internal\\reporting\\export.go'

with open(jsonl_file, 'r', encoding='utf-16') as f:
    lines = f.readlines()

for line in lines:
    if '"targetContent":' in line or '"TargetContent":' in line:
        data = json.loads(line)
        # the tool_calls are inside data['tool_calls']
        for tc in data.get('tool_calls', []):
            if tc['name'] == 'replace_file_content':
                args = tc['args']
                target_content = args.get('TargetContent', '')
                if target_content:
                    with open(target_file, 'r', encoding='utf-8') as tf:
                        current_content = tf.read()
                    
                    # Need to replace the current BuildAdminOwnerOutletXLSX function
                    # It starts at "func BuildAdminOwnerOutletXLSX(" and ends at "\n}\n"
                    # Using regex to find the function block
                    
                    match = re.search(r'func BuildAdminOwnerOutletXLSX\(.*?\)\s*\(\[\]byte,\s*error\)\s*\{.*?\n\}', current_content, re.DOTALL)
                    if match:
                        new_content = current_content[:match.start()] + target_content + current_content[match.end():]
                        with open(target_file, 'w', encoding='utf-8') as tf:
                            tf.write(new_content)
                        print("Replaced successfully!")
                    else:
                        print("Could not find the function to replace")
                    break
        break
