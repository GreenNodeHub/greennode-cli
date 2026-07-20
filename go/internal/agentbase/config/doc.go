// Package config resolves the active environment context and manages credentials
// for the GreenNode AgentBase CLI.
//
// Resolution priority (first wins for each field):
//  1. Environment variable (GREENNODE_ENV, GREENNODE_CLIENT_ID, etc.)
//  2. Corresponding field in ./.greennode.json (current working directory)
//  3. Default value (env defaults to "prod")
package config
