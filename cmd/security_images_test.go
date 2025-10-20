package cmd

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"nudgebee.com/nbctl/pkg/testutil"
)

func TestSecurityImagesCmd(t *testing.T) {
	// Sample GraphQL response data
	sampleResponse := `{
	    "recommendation": {
	        "rows": [
	            {
	                "severity": "Critical",
	                "recommendation": {"VulnerabilityID": "CVE-2022-29155", "PrimaryURL": "https://www.cve.org/CVERecord?id=CVE-2022-29155", "CVSS": {"nvd": {"V2Score": 7.5, "V3Score": 9.8, "V2Vector": "AV:N/AC:L/Au:N/C:P/I:P/A:P", "V3Vector": "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H"}, "redhat": {"V3Score": 6.5, "V3Vector": "CVSS:3.1/AV:N/AC:H/PR:N/UI:N/S:U/C:H/I:L/A:N"}, "bitnami": {"V3Score": 9.8, "V3Vector": "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H"}}, "Layer": {}, "PkgID": "libldap-2.4-2@2.4.47+dfsg-3+deb10u6", "Title": "openldap: OpenLDAP SQL injection", "CweIDs": ["CWE-89"], "Status": "fixed", "PkgName": "libldap-2.4-2", "Severity": "CRITICAL", "VendorIDs": ["DSA-5140-1"], "DataSource": {"ID": "debian", "URL": "https://salsa.debian.org/security-tracker-team/security-tracker", "Name": "Debian Security Tracker"}, "References": ["https://access.redhat.com/security/cve/CVE-2022-29155", "https://bugs.openldap.org/show_bug.cgi?id=9815", "https://lists.debian.org/debian-lts-announce/2022/05/msg00032.html", "https://nvd.nist.gov/vuln/detail/CVE-2022-29155", "https://security.netapp.com/advisory/ntap-20220609-0007/", "https://ubuntu.com/security/notices/USN-5424-1", "https://ubuntu.com/security/notices/USN-5424-2", "https://www.cve.org/CVERecord?id=CVE-2022-29155", "https://www.debian.org/security/2022/dsa-5140"], "image_name": "docker.io/bitnamilegacy/postgresql:11.14.0", "Description": "In OpenLDAP 2.x before 2.5.12 and 2.6.x before 2.6.2, a SQL injection vulnerability exists in the experimental back-sql backend to slapd, via a SQL statement within an LDAP query. This can occur during an LDAP search operation when the search filter is processed, due to a lack of proper escaping.", "FixedVersion": "2.4.47+dfsg-3+deb10u7", "PkgIdentifier": {"UID": "bcb4e2ef274335c9", "PURL": "pkg:deb/debian/libldap-2.4-2@2.4.47%2Bdfsg-3%2Bdeb10u6?arch=amd64&distro=debian-10.11"}, "PublishedDate": "2022-05-10T18:15:00Z"},
	                "image": "docker.io/bitnamilegacy/postgresql:11.14.0",
	                "id": "d690faaf-0d77-4901-abe2-512ba06b71b4",
	                "namespace": "sonarqube",
	                "workload_name": "sonarqube-postgresql",
	                "package_id": "libldap-2.4-2@2.4.47+dfsg-3+deb10u6"
	            }
	        ]
	    },
	    "recommendation_aggregate": {
	        "rows": [
	            {
	                "count": 1
	            }
	        ]
	    }
	}`

	mockData := make(map[string]interface{})
	err := json.Unmarshal([]byte(sampleResponse), &mockData)
	require.NoError(t, err)

	args := []string{
		"security", "list-image-cves",
		"--account-id", "test-account",
		"--limit", "1",
		"--offset", "0",
	}

	got, err := testutil.RunWithSimpleGraphQL(mockData, rootCmd, args)
	assert.NoError(t, err)

	assert.True(t, strings.Contains(got, "Security Recommendations:"))
	assert.True(t, strings.Contains(got, "Severity"))
	assert.True(t, strings.Contains(got, "Image"))
	assert.True(t, strings.Contains(got, "Workload"))
	assert.True(t, strings.Contains(got, "Namespace"))
	assert.True(t, strings.Contains(got, "Package ID"))
	assert.True(t, strings.Contains(got, "CVE-2022-29155"))
	assert.True(t, strings.Contains(got, "Critical"))
	assert.True(t, strings.Contains(got, "docker.io/bitnamilegacy/postgresql:11.14.0"))
	assert.True(t, strings.Contains(got, "sonarqube-postgresql"))
	assert.True(t, strings.Contains(got, "sonarqube"))
	assert.True(t, strings.Contains(got, "libldap-2.4-2@2.4.47+dfsg-3+deb10u6"))
	assert.True(t, strings.Contains(got, "Total Recommendations: 1"))
}
