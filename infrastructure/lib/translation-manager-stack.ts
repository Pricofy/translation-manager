/**
 * Translation Manager Stack
 *
 * Deploys the Go Lambda that orchestrates translation requests.
 * Routes to 4 single-direction translator Lambdas:
 * - translator-romance-en: ES/FR/IT/PT → EN
 * - translator-en-romance: EN → ES/FR/IT/PT
 * - translator-de-en: DE → EN
 * - translator-en-de: EN → DE
 */

import * as cdk from 'aws-cdk-lib';
import * as lambda from 'aws-cdk-lib/aws-lambda';
import * as logs from 'aws-cdk-lib/aws-logs';
import * as iam from 'aws-cdk-lib/aws-iam';
import * as events from 'aws-cdk-lib/aws-events';
import * as targets from 'aws-cdk-lib/aws-events-targets';
import { Construct } from 'constructs';
import * as path from 'path';

export interface TranslationManagerStackProps extends cdk.StackProps {
  environment: 'dev' | 'prod';
}

// The 4 translator Lambdas
const TRANSLATORS = [
  'translator-romance-en',
  'translator-en-romance',
  'translator-de-en',
  'translator-en-de',
];

export class TranslationManagerStack extends cdk.Stack {
  public readonly managerFunction: lambda.Function;

  constructor(scope: Construct, id: string, props: TranslationManagerStackProps) {
    super(scope, id, props);

    const { environment } = props;

    // Lambda function
    this.managerFunction = new lambda.Function(this, 'ManagerFunction', {
      functionName: 'pricofy-translation-manager',
      runtime: lambda.Runtime.PROVIDED_AL2023,
      architecture: lambda.Architecture.ARM_64,
      handler: 'bootstrap',
      code: lambda.Code.fromAsset(path.join(__dirname, '../../dist')),
      timeout: cdk.Duration.seconds(120),
      memorySize: 128,
      environment: {
        ENVIRONMENT: environment,
      },
      description: `Translation orchestrator - routes to translator Lambdas (${environment})`,
    });

    // Grant invoke permissions on all 4 translator Lambdas
    for (const translator of TRANSLATORS) {
      const functionArn = `arn:aws:lambda:${this.region}:${this.account}:function:pricofy-${translator}`;
      this.managerFunction.addToRolePolicy(
        new iam.PolicyStatement({
          actions: ['lambda:InvokeFunction'],
          resources: [functionArn],
        })
      );
    }

    // Log group
    new logs.LogGroup(this, 'ManagerLogGroup', {
      logGroupName: '/aws/lambda/pricofy-translation-manager',
      retention: logs.RetentionDays.ONE_MONTH,
      removalPolicy: cdk.RemovalPolicy.DESTROY,
    });

    // Lambda Warmup (Cold Start Prevention)
    const warmupRule = new events.Rule(this, 'WarmupRule', {
      ruleName: `pricofy-translation-manager-warmup-${environment}`,
      schedule: events.Schedule.rate(cdk.Duration.minutes(5)),
      description: `Warmup for translation-manager (${environment})`,
    });

    warmupRule.addTarget(
      new targets.LambdaFunction(this.managerFunction, {
        event: events.RuleTargetInput.fromObject({
          source: 'warmup',
          concurrency: 2, // Total instances: 1 + 2 = 3
        }),
        retryAttempts: 0,
      })
    );

    // Self-invoke permission (manual ARN to avoid circular dependency)
    const selfFunctionArn = `arn:aws:lambda:${this.region}:${this.account}:function:pricofy-translation-manager`;
    this.managerFunction.addToRolePolicy(
      new iam.PolicyStatement({
        effect: iam.Effect.ALLOW,
        actions: ['lambda:InvokeFunction'],
        resources: [selfFunctionArn],
      })
    );

    // Outputs
    new cdk.CfnOutput(this, 'ManagerFunctionArn', {
      value: this.managerFunction.functionArn,
      exportName: `Pricofy-TranslationManager-${environment}`,
    });

    new cdk.CfnOutput(this, 'ManagerFunctionName', {
      value: this.managerFunction.functionName,
    });

    // Tags
    cdk.Tags.of(this).add('Project', 'Pricofy');
    cdk.Tags.of(this).add('Environment', environment);
    cdk.Tags.of(this).add('Component', 'Translation-Manager');
    cdk.Tags.of(this).add('ManagedBy', 'CDK');
  }
}
