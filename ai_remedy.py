import json
import os
import sys
import google.generativeai as genai

def remediate():
    # 1. Load scan results
    try:
        with open('trivy-results.json', 'r') as f:
            scan_data = json.load(f)
    except Exception as e:
        print(f"Error loading scan results: {e}")
        return

    vulnerabilities = []
    if 'Results' in scan_data:
        for result in scan_data['Results']:
            if 'Vulnerabilities' in result:
                for vuln in result['Vulnerabilities']:
                    vulnerabilities.append({
                        'PkgName': vuln.get('PkgName'),
                        'InstalledVersion': vuln.get('InstalledVersion'),
                        'FixedVersion': vuln.get('FixedVersion'),
                        'ID': vuln.get('VulnerabilityID'),
                        'Title': vuln.get('Title')
                    })

    if not vulnerabilities:
        print("No vulnerabilities found.")
        return

    # 2. Get the content of requirements.txt
    try:
        with open('requirements.txt', 'r') as f:
            reqs_content = f.read()
    except Exception as e:
        print(f"Error reading requirements.txt: {e}")
        reqs_content = ""

    # 3. Setup Gemini AI
    api_key = os.getenv('GEMINI_API_KEY')
    if not api_key:
        print("GEMINI_API_KEY not found. Attempting simple pattern match remediation...")
        # Fallback to simple replacement if fixed version is known
        new_content = reqs_content
        for v in vulnerabilities:
            if v['FixedVersion'] and v['PkgName']:
                pkg = v['PkgName']
                fixed = v['FixedVersion']
                # Very basic replacement logic
                import re
                new_content = re.sub(rf"{pkg}==[\d.]+", f"{pkg}=={fixed}", new_content)
        
        with open('requirements.txt', 'w') as f:
            f.write(new_content)
        print("Applied simple pattern match fixes.")
        return

    genai.configure(api_key=api_key)
    model = genai.GenerativeModel('gemini-1.5-flash')

    prompt = f"""
    You are a DevSecOps AI. I have a security scan result from Trivy and my current requirements.txt.
    Please update the requirements.txt to fix the vulnerabilities.

    Trivy Found:
    {json.dumps(vulnerabilities, indent=2)}

    Current requirements.txt:
    {reqs_content}

    Instructions:
    1. Update versions of packages to at least the 'FixedVersion' mentioned.
    2. Maintain the format of the original file.
    3. Return ONLY the content of the updated requirements.txt file. No markdown, no explanation.
    """

    response = model.generate_content(prompt)
    if response and response.text:
        new_reqs = response.text.strip()
        # Clean up in case Gemini adds ```python or ```
        if "```" in new_reqs:
            new_reqs = new_reqs.split("```")[-2]
            if new_reqs.startswith("python") or new_reqs.startswith("text"):
                new_reqs = "\n".join(new_reqs.split("\n")[1:])

        with open('requirements.txt', 'w') as f:
            f.write(new_reqs)
        print("AI-generated fixes applied to requirements.txt")
    else:
        print("AI failed to generate a response.")

if __name__ == "__main__":
    remediate()
