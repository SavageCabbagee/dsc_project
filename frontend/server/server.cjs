const express = require('express');
const { exec } = require('child_process');
const cors = require('cors');

const app = express();
app.use(express.json());
app.use(cors());

app.post('/api/start-container', (req, res) => {
  console.log("CALLED");
  const { containerName } = req.body;
  console.log(containerName)

  if (!containerName) {
    return res.status(400).send({ error: 'Container name is required' });
  }
  console.log(`Executing: docker start ${containerName}-1`);
  exec(`docker start backend-${containerName}-1`, (error, stdout, stderr) => {
    if (error) {
      console.log(error)
      return res.status(500).send({ error: stderr || error.message });
    }
    res.send({ message: `Container ${containerName} started`, output: stdout });
  });
});

const PORT = 3009; // Backend port
app.listen(PORT, () => {
  console.log(`Backend running on http://localhost:${PORT}`);
});