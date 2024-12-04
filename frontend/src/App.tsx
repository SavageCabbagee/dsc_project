import { useState, useEffect, useRef } from "react";
import { Button } from "./components/button";
import { Input } from "./components/input";
import { Card, CardContent, CardHeader, CardTitle } from "./components/card";

type NodeStatus = {
  aliveState: boolean;
  nodeId: number;
  state: number;
  currentTerm: number;
  votedFor: number;
  leaderId: number;
  lastLogIndex: number;
  lastLogTerm: number;
  logs: Array<{
    term: number;
    command: { operation: string; key: string; value: string };
  }>;
  store: { [key: string]: string };
};

const nodeAddresses = [
  "http://localhost:11001",
  "http://localhost:11002",
  "http://localhost:11003",
  "http://localhost:11004",
  "http://localhost:11005",
  "http://localhost:11006",
  "http://localhost:11007",
  "http://localhost:11008",
  "http://localhost:11009",

];

const stateMap = ["Follower", "Candidate", "Leader"];

export default function Component() {
  const [nodeStatuses, setNodeStatuses] = useState<NodeStatus[]>([]);
  const [keyValues, setKeyValues] = useState<{
    [key: number]: { key: string; value: string };
  }>({
    1: { key: "", value: "" },
    2: { key: "", value: "" },
    3: { key: "", value: "" },
    4: { key: "", value: "" },
    5: { key: "", value: "" },
    6: { key: "", value: "" },
    7: { key: "", value: "" },
    8: { key: "", value: "" },
    9: { key: "", value: "" },
  });
  interface FaultSimulator {
    id: number;
    name: string;
  }
  const [currentFault, setCurrentFault] = useState(Number)
  const [activeFaultList, setActiveFaultList] = useState<FaultSimulator[]>([]);
  const [isTaskRunning, setIsTaskRunning] = useState<boolean>(false);
  const sessionsRef = useRef<FaultSimulator[]>([]);
  let originalStatus = { nodeId: -99, state: -99, currentTerm: -99, votedFor: -99, leaderId: -99, lastLogIndex: -99, lastLogTerm: -99, logs: [], store: {} }

  useEffect(() => {
    let originalNodeStatuses = [];
    for (const key in keyValues) {
      originalNodeStatuses.push({ ...originalStatus, nodeId: key })
    }
  }, [])

  useEffect(() => {
    const fetchStatuses = async () => {
      const statuses = await Promise.all(
        nodeAddresses.map((address, index) =>
          fetch(`${address}/status`)
            .then((res) => res.json())
            .catch(() => ({ ...originalStatus, nodeId: index + 1 }))
          // .catch(() => null)
        )
      );
      setNodeStatuses(statuses.filter(Boolean));
    };

    fetchStatuses();
    const interval = setInterval(fetchStatuses, 1000);
    return () => clearInterval(interval);
  }, []);

  const handleGet = async (nodeId: number) => {
    const { key } = keyValues[nodeId];
    const response = await fetch(`${nodeAddresses[nodeId - 1]}/key/${key}`);
    const data = await response.json();
    setKeyValues((prev) => ({
      ...prev,
      [nodeId]: { ...prev[nodeId], value: data.value || "Not found" },
    }));
  };

  const handleSet = async (nodeId: number) => {
    const { key, value } = keyValues[nodeId];
    await fetch(`${nodeAddresses[nodeId - 1]}/key`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ [key]: value }),
    });
    // Optionally, you can fetch the updated value here
  };

  const handleDelete = async (nodeId: number) => {
    const { key } = keyValues[nodeId];
    await fetch(`${nodeAddresses[nodeId - 1]}/key/${key}`, {
      method: "DELETE",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ key }),
    });
    setKeyValues((prev) => ({
      ...prev,
      [nodeId]: { ...prev[nodeId], value: "" },
    }));
  };

  const handleKill = async (nodeId: number) => {
    await fetch(`${nodeAddresses[nodeId - 1]}/kill`, { method: "POST" });
  };

  const handleStart = async (nodeId: number) => {
    await fetch(`http://localhost:3009/api/start-container`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json", // Specify the content type as JSON
      },
      body: JSON.stringify({
        // Add the data you want to send in the body
        containerName: `node${nodeId}`,
      }),
    });
  };
  const handleCloseSession = (faultId: number) => {
    setActiveFaultList((prevFault) =>
      prevFault.filter((fault) => fault.id !== faultId)
    );
  };
  const handleAddSession = () => {

    // Validate input
    if (currentFault > 9 || currentFault < 1) {
      alert("Node Doesn't Exist!");
      return;
    }
    if (activeFaultList.map((x) => { return x.id }).indexOf(currentFault) != -1) {
      alert("Node Already Under Intermittent Fault");
      return;
    }

    // Add the new session to the list
    setActiveFaultList((prevFault) => [
      ...prevFault,
      { id: currentFault, name: "Node " + currentFault.toString() },
    ]);

  };
  const simulateFault = async (fault: FaultSimulator) => {
    console.log(`Started Fault loop for ${fault.name}`);
    const sleep = (delay: number) => new Promise((resolve) => setTimeout(resolve, delay))
    // Simulate async task, e.g., waiting for a timeout
    var sleepTime = (Math.random() * 7) + 4
    var waitTime = 10 - sleepTime
    await sleep(waitTime * 1000)
    handleKill(fault.id)

    await sleep(sleepTime * 1000)
    handleStart(fault.id)

    console.log(`Completed Loop for ${fault.name}`);
  };

  const runActiveFaults = async () => {
    for (const fault of sessionsRef.current) {
      simulateFault(fault); // Run tasks sequentially
      // If you want them to run concurrently, you can use:
    }
  };
  useEffect(() => {
    sessionsRef.current = activeFaultList;
  }, [activeFaultList]);

  const startBackgroundTasks = async () => {
    if (isTaskRunning) return; // Prevent multiple intervals from starting
    setIsTaskRunning(true);

    while (true) {
      if (sessionsRef.current.length === 0) {
        console.log("No active sessions. Waiting...");
        await new Promise((resolve) => setTimeout(resolve, 11000)); // Wait before checking again
        continue;
      }

      runActiveFaults();


      await new Promise((resolve) => setTimeout(resolve, 11000)); // Wait 10 seconds before next cycle
      console.log("Completed cycle. Running next cycle...");
    }
  };
  startBackgroundTasks()




  return (
    <div className='container mx-auto p-4'>
      <h1 className='text-2xl font-bold mb-4'>Raft Cluster Status</h1>
      <div className='grid grid-cols-1 md:grid-cols-5 gap-4 mb-4'>
        {nodeStatuses.map((status) => (
          <Card
            key={status.nodeId}
            className={`${status.state != -99 ? "bg-green-200" : "bg-red-200"} ${status.state === 2 ? "border-primary bg-blue-200" : ""}`}
          >
            <CardHeader>
              <CardTitle>Node {status.nodeId}</CardTitle>
            </CardHeader>
            <CardContent>
              <div className='space-y-2 mb-4'>
                <p>
                  <span className='font-semibold'>State:</span>{" "}
                  {stateMap[status.state]}
                </p>
                <p>
                  <span className='font-semibold'>Term:</span>{" "}
                  {status.currentTerm}
                </p>
                <p>
                  <span className='font-semibold'>Voted For:</span>{" "}
                  {status.votedFor}
                </p>
                <p>
                  <span className='font-semibold'>Leader ID:</span>{" "}
                  {status.leaderId}
                </p>
                <p>
                  <span className='font-semibold'>Last Log Index:</span>{" "}
                  {status.lastLogIndex}
                </p>
                <p>
                  <span className='font-semibold'>Last Log Term:</span>{" "}
                  {status.lastLogTerm}
                </p>

                {/* Add Store Display */}
                <div className='mt-4'>
                  <p className='font-semibold mb-2'>Key-Value Store:</p>
                  <div className='max-h-40 overflow-y-auto border rounded p-2'>
                    {Object.entries(status.store || {}).map(([key, value]) => (
                      <div key={key} className='text-sm py-1'>
                        <span className='font-medium'>{key}:</span> {value}
                      </div>
                    ))}
                  </div>
                </div>
              </div>

              <div className='space-y-2'>
                <p className='font-semibold mt-4 mb-2'>Operations:</p>
                <text>Key</text>
                <Input
                  id={`key-${status.nodeId}`}
                  value={keyValues[status.nodeId].key}
                  onChange={(e) =>
                    setKeyValues((prev) => ({
                      ...prev,
                      [status.nodeId]: {
                        ...prev[status.nodeId],
                        key: e.target.value,
                      },
                    }))
                  }
                  placeholder='Enter key'
                />
                <text>Value</text>
                <Input
                  id={`value-${status.nodeId}`}
                  value={keyValues[status.nodeId].value}
                  onChange={(e) =>
                    setKeyValues((prev) => ({
                      ...prev,
                      [status.nodeId]: {
                        ...prev[status.nodeId],
                        value: e.target.value,
                      },
                    }))
                  }
                  placeholder='Enter value'
                />
                <div className='flex space-x-2'>
                  <Button onClick={() => handleGet(status.nodeId)}>Get</Button>
                  <Button onClick={() => handleSet(status.nodeId)}>Set</Button>
                  <Button onClick={() => handleDelete(status.nodeId)}>
                    Delete
                  </Button>
                </div>
              </div>
              <div className='mt-4'>
                <Button
                  onClick={() => handleKill(status.nodeId)}
                  variant='destructive'
                >
                  Kill Node
                </Button>
                <Button
                  onClick={() => handleStart(status.nodeId)}
                  variant='default'
                  className="bg-green-600"
                >
                  Start Node
                </Button>
              </div>
            </CardContent>
          </Card>
        ))}

      </div>
      <div className="block">
        <div className="flex flex-col justify-center align-middle w-1/3 m-auto text-center">
          <h2 className="text-base font-bold mb-4">Node Number</h2>
          <Input
            className=" w-auto my-3"
            id="intermittentFault"
            type="number"
            value={Number(currentFault).toString()}
            onChange={(e) => {
              const sanitizedValue = parseInt(e.target.value, 10);
              setCurrentFault(Number(sanitizedValue))
            }}
          >
          </Input>
          <Button className="bg-green-600 my-3"
            onClick={() => handleAddSession()}>
            Intermittent Faults
          </Button>



        </div>
        <div className="p-4">
          <h1 className="text-lg font-bold mb-4">Active Fault Simulations</h1>
          <div className="flex flex-wrap gap-4">
            {activeFaultList.map((fault) => (
              <div
                key={fault.id}
                className="flex items-center justify-between bg-gray-200 p-4 rounded-md shadow-md w-64"
              >
                <span className="text-gray-700 font-bold">{fault.name}</span>

                <button
                  onClick={() => handleCloseSession(fault.id)}
                  className="text-red-500 hover:text-red-700 font-bold"
                >
                  ×
                </button>
              </div>
            ))}
          </div>
          {activeFaultList.length === 0 && (
            <p className="text-gray-500 mt-4">No active sessions.</p>
          )}
        </div>
      </div>
    </div>
  );
}
