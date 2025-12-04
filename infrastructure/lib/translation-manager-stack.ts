/**
 * Translation Manager Stack
 *
 * Deploys the Go Lambda that orchestrates translation requests.
 * Routes to translator-{src}-{tgt} Lambdas, handles chunking.
 */

import * as cdk from 'aws-cdk-lib';
import * as lambda from 'aws-cdk-lib/aws-lambda';
import * as logs from 'aws-cdk-lib/aws-logs';
import * as iam from 'aws-cdk-lib/aws-iam';
import { Construct } from 'constructs';
import * as path from 'path';

export interface TranslationManagerStackProps extends cdk.StackProps {
  environment: 'dev' | 'prod';
}

// Supported language pairs (without English)
const LANGUAGES = ['es', 'it', 'pt', 'fr', 'de'];

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
      timeout: cdk.Duration.seconds(60),
      memorySize: 128,
      environment: {
        ENVIRONMENT: environment,
      },
      description: `Translation orchestrator - routes to translator Lambdas (${environment})`,
    });

    // Grant invoke permissions on all translator Lambdas
    for (const source of LANGUAGES) {
      for (const target of LANGUAGES) {
        if (source !== target) {
          const functionArn = `arn:aws:lambda:${this.region}:${this.account}:function:pricofy-translator-${source}-${target}`;
          this.managerFunction.addToRolePolicy(
            new iam.PolicyStatement({
              actions: ['lambda:InvokeFunction'],
              resources: [functionArn],
            })
          );
        }
      }
    }

    // Log group
    new logs.LogGroup(this, 'ManagerLogGroup', {
      logGroupName: '/aws/lambda/pricofy-translation-manager',
      retention: logs.RetentionDays.ONE_MONTH,
      removalPolicy: cdk.RemovalPolicy.DESTROY,
    });

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
