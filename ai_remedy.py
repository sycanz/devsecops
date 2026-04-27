import json
import os
import google.generativeai as genai

def load_vulnerabilities(filename):
    """Loads and parses Trivy JSON reports (FS or Config/IaC)."""
    if not os.path.exists(filename):
        return []
    
    with open(filename, 'r') as f:
        data = json.load(f)
    
    vulnerabilities = []
    if 'Results' in data:
        for result in data['Results']:
            target = result.get('Target') # The file path (e.g. terraform/main.tf)
            
            # 1. Capture Dependency Vulnerabilities
            if 'Vulnerabilities' in result:
                for v in result['Vulnerabilities']:
                    vulnerabilities.append({
                        'Target': target,
                        'PkgName': v.get('PkgName'),
                        'FixedVersion': v.get('FixedVersion'),
                        'ID': v.get('VulnerabilityID'),
                        'Title': v.get('Title'),
                        'Type': 'dependency'
                    })
            
            # 2. Capture Infrastructure Misconfigurations
            if 'Misconfigurations' in result:
                for m in result['Misconfigurations']:
                    vulnerabilities.append({
                        'Target': target,
                        'ID': m.get('ID'),
                        'Title': m.get('Title'),
                        'Message': m.get('Message'),
                        'Resolution': m.get('Resolution'),
                        'Type': 'misconfiguration'
                    })
    return vulnerabilities

def remediate():
    # 1. Collect all vulnerabilities from both scan types
    fs_vulns = load_vulnerabilities('trivy-fs-results.json')
    iac_vulns = load_vulnerabilities('trivy-iac-results.json')
    all_vulns = fs_vulns + iac_vulns

    if not all_vulns:
        print("No vulnerabilities found to fix.")
        return

    # 2. Group findings by the file they belong to
    files_to_fix = {}
    for v in all_vulns:
        target = v['Target']
        if target not in files_to_fix:
            files_to_fix[target] = []
        files_to_fix[target].append(v)

    # 3. Setup Gemini AI
    api_key = os.getenv('GEMINI_API_KEY')
    if not api_key:
        print("GEMINI_API_KEY not found. Automated AI remediation requires an API key.")
        return

    genai.configure(api_key=api_key)
    model = genai.GenerativeModel('gemini-1.5-flash')

    # 4. Iterate through each broken file and ask AI for a fixed version
    for target_file, vulns in files_to_fix.items():
        if not os.path.exists(target_file):
            print(f"Skipping fix for {target_file}: File not found in workspace.")
            continue

        print(f"Applying AI logic to secure: {target_file}...")
        with open(target_file, 'r') as f:
            content = f.read()

        prompt = f"""
        You are a Senior DevSecOps Engineer. Your task is to fix the security issues in the provided file.
        
        File Path: {target_file}
        Current Content:
        ---
        {content}
        ---

        Security Vulnerabilities/Misconfigurations detected by Trivy:
        {json.dumps(vulns, indent=2)}

        Instructions:
        1. Rewrite the file to fix the vulnerabilities (update versions) or misconfigurations (fix settings).
        2. Keep the original intent and functionality of the code.
        3. Return ONLY the complete, corrected content of the file. No markdown code blocks (like ```terraform), no explanations.
        """

        response = model.generate_content(prompt)
        if response and response.text:
            fixed_content = response.text.strip()
            
            # Clean up potential markdown formatting from AI response
            if "```" in fixed_content:
                lines = fixed_content.split("\n")
                if lines[0].startswith("```"): lines = lines[1:]
                if lines[-1].startswith("```"): lines = lines[:-1]
                fixed_content = "\n".join(lines).strip()

            with open(target_file, 'w') as f:
                f.write(fixed_content)
            print(f"✅ Fixed {target_file}")
        else:
            print(f"❌ Failed to get a suggestion for {target_file}")

if __name__ == "__main__":
    remediate()
