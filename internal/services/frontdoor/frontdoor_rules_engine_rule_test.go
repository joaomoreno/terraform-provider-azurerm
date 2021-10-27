package frontdoor_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-provider-azurerm/internal/acceptance"
	"github.com/hashicorp/terraform-provider-azurerm/internal/acceptance/check"
)

func TestAccFrontDoorRulesEngine_complete(t *testing.T) {
	data := acceptance.BuildTestData(t, "azurerm_frontdoor", "test")
	r := FrontDoorResource{}

	data.ResourceTest(t, r, []acceptance.TestStep{
		{
			Config: r.deploy(data),
			Check: acceptance.ComposeTestCheckFunc(
				check.That(data.ResourceName).ExistsInAzure(r),
			),
		},
		data.ImportStep("explicit_resource_order"),
	})
}

func (FrontDoorResource) template(data acceptance.TestData) string {
	return fmt.Sprintf(`

provider "azurerm" {
  features {}
}

data "azurerm_client_config" "current" {
}

resource "azurerm_resource_group" "test" {
  name     = "testaccRG-%[1]d"
  location = "%[2]s"
}

resource "azurerm_frontdoor" "test" {
	name                                         = "acctest-FD-%[1]d"
	resource_group_name                          = azurerm_resource_group.test.name
	enforce_backend_pools_certificate_name_check = false
  
	routing_rule {
	  name               = "test-routing-rule"
	  accepted_protocols = ["Http", "Https"]
	  patterns_to_match  = ["/*"]
	  frontend_endpoints = ["exampleFrontendEndpoint1"]
	  
	  forwarding_configuration {
		forwarding_protocol = "MatchRequest"
		backend_pool_name   = "exampleBackendBing"
		cache_enabled 		= true
		cache_duration 		= "P180D"
	  }
	}
  
	backend_pool_load_balancing {
	  name = "exampleLoadBalancingSettings1"
	}
  
	backend_pool_health_probe {
	  name = "exampleHealthProbeSetting1"
	}
  
	backend_pool {
	  name = "exampleBackendBing"
	  backend {
		host_header = "www.bing.com"
		address     = "www.bing.com"
		http_port   = 80
		https_port  = 443
	  }
  
	  load_balancing_name = "exampleLoadBalancingSettings1"
	  health_probe_name   = "exampleHealthProbeSetting1"
	}
  
	frontend_endpoint {
		name      = "exampleFrontendEndpoint1"
		host_name = "acctest-FD-%[1]d.azurefd.net"
	}
  }
  
  resource "azurerm_frontdoor_rules_engine" "test" {
	name                = "corsrules"
	frontdoor_name      = azurerm_frontdoor.test.name
	resource_group_name = azurerm_frontdoor.test.resource_group_name
  
	rule {
	  name = "debug"
	  priority = 1
  
	  match_condition {
		variable = "RequestMethod"
		operator = "Equal"
		value = ["GET", "POST"]
	  }
  
	  action {
		response_header {
		  header_action_type = "Append"
		  header_name        = "X-TEST-HEADER"
		  value              = "CORS CMS Rule"
		}
	  }
	}
  
	rule {
	  name     = "origin"
	  priority = 2
  
	  action {
		request_header {
		  header_action_type = "Overwrite"
		  header_name        = "Origin"
		  value              = "this is an append test"
		}
		response_header {
		  header_action_type = "Overwrite"
		  header_name        = "Access-Control-Allow-Origin"
		  value              = "*"
		}
	  }
	}
  
	depends_on = [
	  azurerm_frontdoor.test
	]
  }

`, data.RandomInteger, data.Locations.Primary)
}

func (FrontDoorResource) associate(data acceptance.TestData) string {
	return fmt.Sprintf(`

	# before the rules engine config can be deleted it needs to be unassociated
	resource "null_resource" "delete_association" {
		triggers = {
			frontdoor =	azurerm_frontdoor.test.name
			resourcegroup = azurerm_resource_group.test.name
			subscriptionid = data.azurerm_client_config.current.subscription_id
		}

		provisioner "local-exec" {
		  when = destroy
		  command = "az account set --subscription ${self.triggers.subscriptionid} && az config set extension.use_dynamic_install=yes_without_prompt && az network front-door routing-rule update --front-door-name ${self.triggers.frontdoor} --resource-group ${self.triggers.resourcegroup} --name test-routing-rule --remove rulesEngine"
		}
	
		depends_on = [
		  azurerm_frontdoor_rules_engine.test
		]
	  }

	# the rules engine configuration needs to be associated via azurecli today
	resource "null_resource" "test_association" {
		triggers = {
		  rulesengine = azurerm_frontdoor_rules_engine.test.name # only a new name will trigger a new execution
		}

		provisioner "local-exec" {
		  command = "az account set --subscription $ARM_SUBSCRIPTION_ID && az config set extension.use_dynamic_install=yes_without_prompt && az network front-door routing-rule update --front-door-name ${azurerm_frontdoor.test.name} --resource-group ${azurerm_resource_group.test.name} --name test-routing-rule --rules-engine ${self.triggers.rulesengine}"
		}
	
		depends_on = [
		  azurerm_frontdoor.test,
		  azurerm_frontdoor_rules_engine.test
		]
	  }
  
	  `)
}

func (r FrontDoorResource) deploy(data acceptance.TestData) string {
	return fmt.Sprintf(`

	%[1]s

	%[2]s
	`, r.template(data), r.associate(data))
}
