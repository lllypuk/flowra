#!/bin/bash
#
# Keycloak Setup Script for Flowra
#
# This script waits for Keycloak to be ready and imports the flowra realm
# configuration. It can be used for initial setup or to reset the realm.
#
# Usage:
#   ./scripts/setup-keycloak.sh [options]
#
# Options:
#   --reset    Delete existing realm before import
#   --wait     Only wait for Keycloak, don't import
#   --help     Show this help message
#
set -e

# Configuration
KEYCLOAK_URL="${KEYCLOAK_URL:-http://localhost:8090}"
REALM="flowra"
ADMIN_USERNAME="${KEYCLOAK_ADMIN:-admin}"
ADMIN_PASSWORD="${KEYCLOAK_ADMIN_PASSWORD:-admin123}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
REALM_EXPORT_FILE="${PROJECT_ROOT}/configs/keycloak/realm-export.json"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Show help
show_help() {
    echo "Keycloak Setup Script for Flowra"
    echo ""
    echo "Usage: $0 [options]"
    echo ""
    echo "Options:"
    echo "  --reset    Delete existing realm before import"
    echo "  --wait     Only wait for Keycloak to be ready, don't import"
    echo "  --help     Show this help message"
    echo ""
    echo "Environment variables:"
    echo "  KEYCLOAK_URL              Keycloak base URL (default: http://localhost:8090)"
    echo "  KEYCLOAK_ADMIN            Admin username (default: admin)"
    echo "  KEYCLOAK_ADMIN_PASSWORD   Admin password (default: admin123)"
}

# Wait for Keycloak to be ready
wait_for_keycloak() {
    log_info "Waiting for Keycloak to start at ${KEYCLOAK_URL}..."

    local max_attempts=60
    local attempt=1

    while [ $attempt -le $max_attempts ]; do
        if curl -sf "${KEYCLOAK_URL}/health/ready" > /dev/null 2>&1; then
            log_info "Keycloak is ready!"
            return 0
        fi

        echo -n "."
        sleep 2
        attempt=$((attempt + 1))
    done

    echo ""
    log_error "Keycloak failed to start after $((max_attempts * 2)) seconds"
    return 1
}

# Get admin access token
get_admin_token() {
    log_info "Obtaining admin access token..."

    local response
    response=$(curl -sf -X POST "${KEYCLOAK_URL}/realms/master/protocol/openid-connect/token" \
        -H "Content-Type: application/x-www-form-urlencoded" \
        -d "username=${ADMIN_USERNAME}" \
        -d "password=${ADMIN_PASSWORD}" \
        -d "grant_type=password" \
        -d "client_id=admin-cli" 2>&1)

    if [ $? -ne 0 ]; then
        log_error "Failed to get admin token. Check admin credentials."
        return 1
    fi

    ADMIN_TOKEN=$(echo "$response" | grep -o '"access_token":"[^"]*"' | cut -d'"' -f4)

    if [ -z "$ADMIN_TOKEN" ]; then
        log_error "Failed to parse admin token from response"
        return 1
    fi

    log_info "Admin token obtained successfully"
}

# Check if realm exists
realm_exists() {
    local status_code
    status_code=$(curl -sf -o /dev/null -w "%{http_code}" \
        -H "Authorization: Bearer ${ADMIN_TOKEN}" \
        "${KEYCLOAK_URL}/admin/realms/${REALM}")

    [ "$status_code" = "200" ]
}

# Delete existing realm
delete_realm() {
    log_info "Deleting existing realm '${REALM}'..."

    local status_code
    status_code=$(curl -sf -o /dev/null -w "%{http_code}" \
        -X DELETE \
        -H "Authorization: Bearer ${ADMIN_TOKEN}" \
        "${KEYCLOAK_URL}/admin/realms/${REALM}")

    if [ "$status_code" = "204" ]; then
        log_info "Realm deleted successfully"
        return 0
    else
        log_error "Failed to delete realm (HTTP ${status_code})"
        return 1
    fi
}

# Import realm from JSON file
import_realm() {
    if [ ! -f "$REALM_EXPORT_FILE" ]; then
        log_error "Realm export file not found: ${REALM_EXPORT_FILE}"
        return 1
    fi

    log_info "Importing realm from ${REALM_EXPORT_FILE}..."

    local status_code
    local response
    response=$(curl -sf -w "\n%{http_code}" \
        -X POST "${KEYCLOAK_URL}/admin/realms" \
        -H "Authorization: Bearer ${ADMIN_TOKEN}" \
        -H "Content-Type: application/json" \
        -d @"${REALM_EXPORT_FILE}" 2>&1)

    status_code=$(echo "$response" | tail -n1)

    if [ "$status_code" = "201" ]; then
        log_info "Realm imported successfully!"
        return 0
    elif [ "$status_code" = "409" ]; then
        log_warn "Realm already exists. Use --reset to delete and reimport."
        return 0
    else
        log_error "Failed to import realm (HTTP ${status_code})"
        echo "$response" | head -n -1
        return 1
    fi
}

# Verify realm setup
verify_realm() {
    log_info "Verifying realm configuration..."

    # Check realm exists
    if ! realm_exists; then
        log_error "Realm '${REALM}' does not exist"
        return 1
    fi
    log_info "✓ Realm '${REALM}' exists"

    # Check client exists
    local client_response
    client_response=$(curl -sf \
        -H "Authorization: Bearer ${ADMIN_TOKEN}" \
        "${KEYCLOAK_URL}/admin/realms/${REALM}/clients?clientId=flowra-backend")

    if echo "$client_response" | grep -q "flowra-backend"; then
        log_info "✓ Client 'flowra-backend' exists"
    else
        log_warn "Client 'flowra-backend' not found"
    fi

    # Check roles exist
    local roles_response
    roles_response=$(curl -sf \
        -H "Authorization: Bearer ${ADMIN_TOKEN}" \
        "${KEYCLOAK_URL}/admin/realms/${REALM}/roles")

    for role in "user" "admin" "workspace_owner" "workspace_admin"; do
        if echo "$roles_response" | grep -q "\"name\":\"${role}\""; then
            log_info "✓ Role '${role}' exists"
        else
            log_warn "Role '${role}' not found"
        fi
    done

    # Check groups exist
    local groups_response
    groups_response=$(curl -sf \
        -H "Authorization: Bearer ${ADMIN_TOKEN}" \
        "${KEYCLOAK_URL}/admin/realms/${REALM}/groups")

    for group in "users" "admins"; do
        if echo "$groups_response" | grep -q "\"name\":\"${group}\""; then
            log_info "✓ Group '${group}' exists"
        else
            log_warn "Group '${group}' not found"
        fi
    done

    # Check test users exist
    local users_response
    users_response=$(curl -sf \
        -H "Authorization: Bearer ${ADMIN_TOKEN}" \
        "${KEYCLOAK_URL}/admin/realms/${REALM}/users")

    for user in "testuser" "admin" "alice" "bob"; do
        if echo "$users_response" | grep -q "\"username\":\"${user}\""; then
            log_info "✓ User '${user}' exists"
        else
            log_warn "User '${user}' not found"
        fi
    done

    log_info "Realm verification complete!"
}

# Print summary
print_summary() {
    echo ""
    echo "============================================"
    echo "Keycloak Setup Complete!"
    echo "============================================"
    echo ""
    echo "Keycloak Admin Console: ${KEYCLOAK_URL}/admin"
    echo "Admin credentials: ${ADMIN_USERNAME} / ${ADMIN_PASSWORD}"
    echo ""
    echo "Realm: ${REALM}"
    echo "Client ID: flowra-backend"
    echo "Client Secret: flowra-dev-secret-change-in-production"
    echo ""
    echo "Test users:"
    echo "  testuser / password123 (role: user)"
    echo "  admin / admin123 (roles: user, admin)"
    echo "  alice / password123 (roles: user, workspace_owner)"
    echo "  bob / password123 (role: user)"
    echo ""
}

# Main script
main() {
    local do_reset=false
    local wait_only=false

    # Parse arguments
    while [ $# -gt 0 ]; do
        case "$1" in
            --reset)
                do_reset=true
                shift
                ;;
            --wait)
                wait_only=true
                shift
                ;;
            --help|-h)
                show_help
                exit 0
                ;;
            *)
                log_error "Unknown option: $1"
                show_help
                exit 1
                ;;
        esac
    done

    # Wait for Keycloak
    wait_for_keycloak || exit 1

    if [ "$wait_only" = true ]; then
        log_info "Keycloak is ready. Exiting (--wait mode)."
        exit 0
    fi

    # Get admin token
    get_admin_token || exit 1

    # Handle reset if requested
    if [ "$do_reset" = true ] && realm_exists; then
        delete_realm || exit 1
    fi

    # Import realm
    import_realm || exit 1

    # Verify configuration
    verify_realm

    # Print summary
    print_summary
}

# Run main function
main "$@"
