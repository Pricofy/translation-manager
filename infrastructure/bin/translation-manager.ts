#!/usr/bin/env node
import 'source-map-support/register';
import * as cdk from 'aws-cdk-lib';
import { TranslationManagerStack } from '../lib/translation-manager-stack';

const app = new cdk.App();

const environment = app.node.tryGetContext('environment') || 'dev';

new TranslationManagerStack(app, 'Pricofy-TranslationManager', {
  environment,
  env: {
    account: process.env.CDK_DEFAULT_ACCOUNT,
    region: process.env.CDK_DEFAULT_REGION || 'eu-west-1',
  },
});
