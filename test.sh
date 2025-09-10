#!/bin/bash
# Test script for zopen-mcp-server

# Build the server
echo "Building zopen-mcp-server..."
make build

# Check if build was successful
if [ $? -ne 0 ]; then
    echo "Build failed!"
    exit 1
fi

# Create a test input JSON for the MCP protocol - zopen version
cat > test_input_zopen.json << EOF
{
    "id": "test-1",
    "method": "callTool",
    "params": {
        "name": "zopen_version",
        "args": {}
    }
}
EOF

# Add newline to the end of the JSON
echo "" >> test_input_zopen.json

# Run the server with the test input
echo "Running test with zopen_version..."
cat test_input_zopen.json | ./zopen-mcp-server > test_output_zopen.json

# Check if the server ran successfully
if [ $? -ne 0 ]; then
    echo "Server execution failed for zopen_version!"
    exit 1
fi

# Display the output
echo "Server response for zopen_version:"
cat test_output_zopen.json

# Create a test input JSON for the MCP protocol - zopen-generate help
cat > test_input_generate.json << EOF
{
    "id": "test-2",
    "method": "callTool",
    "params": {
        "name": "zopen_generate_help",
        "args": {}
    }
}
EOF

# Add newline to the end of the JSON
echo "" >> test_input_generate.json

# Run the server with the test input
echo "Running test with zopen_generate_help..."
cat test_input_generate.json | ./zopen-mcp-server > test_output_generate.json

# Check if the server ran successfully
if [ $? -ne 0 ]; then
    echo "Server execution failed for zopen_generate_help!"
    exit 1
fi

# Display the output
echo "Server response for zopen_generate_help:"
cat test_output_generate.json

# Create a more complex test for zopen_generate
cat > test_input_project.json << EOF
{
    "id": "test-3",
    "method": "callTool",
    "params": {
        "name": "zopen_generate",
        "args": {
            "name": "test-project",
            "description": "A test project",
            "categories": "development utilities",
            "license": "MIT",
            "build_line": "stable",
            "force": true
        }
    }
}
EOF

# Add newline to the end of the JSON
echo "" >> test_input_project.json

# Run the server with the test input
echo "Running test with zopen_generate..."
cat test_input_project.json | ./zopen-mcp-server > test_output_project.json

# Check if the server ran successfully
if [ $? -ne 0 ]; then
    echo "Server execution failed for zopen_generate!"
    exit 1
fi

# Display the output
echo "Server response for zopen_generate:"
cat test_output_project.json

# Clean up test files
#rm -f test_input_*.json test_output_*.json

echo "All tests completed successfully!"

# Made with Bob
