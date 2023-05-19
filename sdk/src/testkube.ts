import * as generatedClient from "./generated-client";

type InitTestkubeOptions = {
  url: string;
  apiToken?: string;
};

export const initTestkube = ({ url, apiToken }: InitTestkubeOptions) => {
  generatedClient.defaults.baseUrl = url;
  console.log("Testkube SDK initialized with url: ", url);
};

export const test = (name: string) => {
  return {
    run: () => {
      return generatedClient.executeTest(name, {});
    },

    metrics: generatedClient.getTestMetrics(name),
    runUntilEnd: async () => {
      const createdExecution = await generatedClient.executeTest(name, {});
      if (createdExecution.status !== 201) {
        return createdExecution.data;
      }

      // @ts-ignore
      let currentExecutionStatus = createdExecution.data.executionResult.status;
      let execution;
      while (
        // @ts-ignore
        createdExecution.status === 201 &&
        // @ts-ignore
        currentExecutionStatus === "running"
      ) {
        execution = await generatedClient.getExecutionById(
          // @ts-ignore
          createdExecution.data.id
        );

        if (execution.status !== 200 || !execution.data.executionResult) {
          return execution.data;
        }

        currentExecutionStatus = execution.data.executionResult.status;
      }

      return execution;
    },
  };
};

export * from "./generated-client";
