import pytest

# Licensing, SSO (SAML/OIDC), and the Zeus/subscription integrations ship in the
# enterprise modules (ee/, cmd/enterprise/) that this MIT-only fork removes, so
# anything exercising them is skipped here.
ENTERPRISE_ONLY = pytest.mark.skip(reason="enterprise-only; not part of the MIT-only community build")
