package grule

import (
	"context"
	"errors"

	"github.com/hyperjumptech/grule-rule-engine/ast"
	"github.com/hyperjumptech/grule-rule-engine/builder"
	"github.com/hyperjumptech/grule-rule-engine/engine"
	"github.com/hyperjumptech/grule-rule-engine/pkg"
	"github.com/warpstreamlabs/bento/public/service"
)

func init() {
	err := service.RegisterProcessor(
		"grule",
		service.NewConfigSpec().
			Summary("A Grule rule engine processor that natively handles structured JSON data.").
			Field(service.NewStringField("rule_file").Default("").Description("Path to the GRL rule file. If not provided, rule_string must be used.")).
			Field(service.NewStringField("rule_string").Default("").Description("GRL rule definitions as a string. Overridden by rule_file if both are provided.")).
			Field(service.NewStringField("fact_name").Default("MF").Description("The variable name injected into the rule engine context.")),
		func(conf *service.ParsedConfig, mgr *service.Resources) (service.Processor, error) {

			ruleFile, _ := conf.FieldString("rule_file")
			ruleString, _ := conf.FieldString("rule_string")
			factName, _ := conf.FieldString("fact_name")

			// Validation: Ensure at least one rule source is provided
			if ruleFile == "" && ruleString == "" {
				return nil, errors.New("either 'rule_file' or 'rule_string' must be provided in the configuration")
			}

			knowledgeLibrary := ast.NewKnowledgeLibrary()
			ruleBuilder := builder.NewRuleBuilder(knowledgeLibrary)

			var resource pkg.Resource
			if ruleFile != "" {
				// Load rules from a file
				resource = pkg.NewFileResource(ruleFile)
			} else {
				// Load rules from a string literal
				resource = pkg.NewBytesResource([]byte(ruleString))
			}

			// Build the rule resource into the KnowledgeLibrary
			err := ruleBuilder.BuildRuleFromResource("BentoRules", "1.0.0", resource)
			if err != nil {
				return nil, err
			}

			// Instantiate the KnowledgeBase (Done once during initialization)
			knowledgeBase, err := knowledgeLibrary.NewKnowledgeBaseInstance("BentoRules", "1.0.0")
			if err != nil {
				return nil, err
			}

			mgr.Logger().Infof("Successfully loaded Grule KnowledgeBase (Fact Name: %s)", factName)

			return &gruleJsonProcessor{
				knowledgeBase: knowledgeBase,
				engine:        engine.NewGruleEngine(),
				logger:        mgr.Logger(),
				factName:      factName,
			}, nil
		})

	if err != nil {
		panic(err)
	}
}

type gruleJsonProcessor struct {
	knowledgeBase *ast.KnowledgeBase
	engine        *engine.GruleEngine
	logger        *service.Logger
	factName      string
}

// Process implements the service.Processor interface.
// It applies the Grule rules to each incoming message.
func (g *gruleJsonProcessor) Process(ctx context.Context, msg *service.Message) (service.MessageBatch, error) {
	// Extract the structured payload (e.g., JSON map) from the message
	jsonData, err := msg.AsStructuredMut()
	if err != nil {
		g.logger.Errorf("Failed to parse message as structured data: %v", err)
		return nil, err
	}

	// Create a new data context for this specific message
	dataCtx := ast.NewDataContext()

	// Inject the structured data into the rule engine context
	err = dataCtx.Add(g.factName, jsonData)
	if err != nil {
		g.logger.Errorf("Failed to add fact to data context: %v", err)
		return nil, err
	}

	// Execute the rule engine
	err = g.engine.Execute(dataCtx, g.knowledgeBase)
	if err != nil {
		g.logger.Errorf("Grule execution failed: %v", err)
		return nil, err
	}

	// Set the modified structured data back to the message payload
	msg.SetStructuredMut(jsonData)

	return service.MessageBatch{msg}, nil
}

// Close implements the service.Processor interface.
// It is called when the Bento pipeline shuts down.
func (g *gruleJsonProcessor) Close(ctx context.Context) error {
	// No network connections or file descriptors need to be explicitly closed
	return nil
}
