package ml

import (
	"context"
	"encoding/csv"
	"fmt"
	"math"
	"os"
	"strconv"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/sjwhitworth/golearn/base"
	"github.com/sjwhitworth/golearn/ensemble"
	"github.com/sjwhitworth/golearn/evaluation"
	"github.com/stitts-dev/dfs-sim/shared/pkg/logger"
	"gorgonia.org/gorgonia"
	"gorgonia.org/tensor"
)

// Predictor handles ML model training and inference
type Predictor struct {
	logger        *logrus.Logger
	neuralNetwork *NeuralNetwork
	randomForest  *RandomForestModel
	modelVersion  string
	isInitialized bool
}

// NeuralNetwork represents a Gorgonia-based neural network
type NeuralNetwork struct {
	graph         *gorgonia.ExprGraph
	inputNode     *gorgonia.Node
	outputNode    *gorgonia.Node
	weights       []*gorgonia.Node
	biases        []*gorgonia.Node
	machine       gorgonia.VM
	learnRate     float64
	hiddenLayers  []int
	inputSize     int
	outputSize    int
}

// RandomForestModel wraps GoLearn random forest
type RandomForestModel struct {
	classifier base.Classifier
	dataset    *base.DenseInstances
	features   []string
	isRegression bool
}

// ModelConfig defines configuration for ML models
type ModelConfig struct {
	ModelType      string    `json:"model_type"` // "neural_network", "random_forest", "ensemble"
	HiddenLayers   []int     `json:"hidden_layers"`
	LearningRate   float64   `json:"learning_rate"`
	BatchSize      int       `json:"batch_size"`
	Epochs         int       `json:"epochs"`
	TreeCount      int       `json:"tree_count"`
	MaxDepth       int       `json:"max_depth"`
	ValidationSplit float64  `json:"validation_split"`
	EarlyStoppping bool      `json:"early_stopping"`
}

// TrainingData represents data for model training
type TrainingData struct {
	Features [][]float64 `json:"features"`
	Labels   []float64   `json:"labels"`
	UserIDs  []int       `json:"user_ids"`
}

// PredictionResult contains model prediction output
type PredictionResult struct {
	UserID      int                    `json:"user_id"`
	ModelType   string                 `json:"model_type"`
	Prediction  interface{}            `json:"prediction"`
	Confidence  float64                `json:"confidence"`
	Features    map[string]float64     `json:"features"`
	ModelVersion string                `json:"model_version"`
	Timestamp   time.Time              `json:"timestamp"`
}

// NewPredictor creates a new ML predictor instance
func NewPredictor(config ModelConfig) *Predictor {
	return &Predictor{
		logger:       logger.GetLogger(),
		modelVersion: "1.0",
		isInitialized: false,
	}
}

// InitializeModels initializes both neural network and random forest models
func (p *Predictor) InitializeModels(ctx context.Context, config ModelConfig) error {
	p.logger.WithFields(logrus.Fields{
		"model_type":     config.ModelType,
		"hidden_layers":  config.HiddenLayers,
		"learning_rate":  config.LearningRate,
	}).Info("Initializing ML models")

	// Initialize neural network
	if config.ModelType == "neural_network" || config.ModelType == "ensemble" {
		nn, err := p.initializeNeuralNetwork(config)
		if err != nil {
			return fmt.Errorf("failed to initialize neural network: %w", err)
		}
		p.neuralNetwork = nn
	}

	// Initialize random forest
	if config.ModelType == "random_forest" || config.ModelType == "ensemble" {
		rf, err := p.initializeRandomForest(config)
		if err != nil {
			return fmt.Errorf("failed to initialize random forest: %w", err)
		}
		p.randomForest = rf
	}

	p.isInitialized = true
	p.logger.Info("ML models initialized successfully")
	return nil
}

// TrainModels trains the ML models with provided data
func (p *Predictor) TrainModels(ctx context.Context, data *TrainingData, config ModelConfig) error {
	if !p.isInitialized {
		return fmt.Errorf("models not initialized")
	}

	p.logger.WithFields(logrus.Fields{
		"training_samples": len(data.Features),
		"feature_count":   len(data.Features[0]),
		"epochs":          config.Epochs,
	}).Info("Starting model training")

	// Split data for validation
	trainData, valData := p.splitTrainingData(data, config.ValidationSplit)

	// Train neural network
	if p.neuralNetwork != nil {
		err := p.trainNeuralNetwork(trainData, valData, config)
		if err != nil {
			return fmt.Errorf("neural network training failed: %w", err)
		}
	}

	// Train random forest
	if p.randomForest != nil {
		err := p.trainRandomForest(trainData, config)
		if err != nil {
			return fmt.Errorf("random forest training failed: %w", err)
		}
	}

	p.logger.Info("Model training completed successfully")
	return nil
}

// Predict generates predictions using trained models
func (p *Predictor) Predict(ctx context.Context, features map[string]float64, config ModelConfig) (*PredictionResult, error) {
	if !p.isInitialized {
		return nil, fmt.Errorf("models not initialized")
	}

	result := &PredictionResult{
		Features:     features,
		ModelVersion: p.modelVersion,
		Timestamp:    time.Now(),
		ModelType:    config.ModelType,
	}

	// Convert features to array
	featureArray := p.featuresToArray(features)

	switch config.ModelType {
	case "neural_network":
		prediction, confidence, err := p.predictNeuralNetwork(featureArray)
		if err != nil {
			return nil, err
		}
		result.Prediction = prediction
		result.Confidence = confidence

	case "random_forest":
		prediction, confidence, err := p.predictRandomForest(featureArray)
		if err != nil {
			return nil, err
		}
		result.Prediction = prediction
		result.Confidence = confidence

	case "ensemble":
		// Ensemble prediction combining both models
		nnPred, nnConf, err := p.predictNeuralNetwork(featureArray)
		if err != nil {
			return nil, err
		}
		
		rfPred, rfConf, err := p.predictRandomForest(featureArray)
		if err != nil {
			return nil, err
		}

		// Weighted ensemble based on confidence
		totalWeight := nnConf + rfConf
		if totalWeight > 0 {
			ensemblePred := (nnPred*nnConf + rfPred*rfConf) / totalWeight
			result.Prediction = ensemblePred
			result.Confidence = (nnConf + rfConf) / 2.0
		}

	default:
		return nil, fmt.Errorf("unsupported model type: %s", config.ModelType)
	}

	return result, nil
}

// initializeNeuralNetwork creates and initializes a neural network
func (p *Predictor) initializeNeuralNetwork(config ModelConfig) (*NeuralNetwork, error) {
	// Create computation graph
	g := gorgonia.NewGraph()
	
	// Determine network architecture
	inputSize := 50 // Default feature count - will be adjusted based on actual data
	outputSize := 1  // Regression output
	hiddenLayers := config.HiddenLayers
	if len(hiddenLayers) == 0 {
		hiddenLayers = []int{64, 32, 16} // Default architecture
	}

	// Input node
	inputNode := gorgonia.NewMatrix(g, tensor.Float64,
		gorgonia.WithShape(-1, inputSize),
		gorgonia.WithName("input"))

	// Build network layers
	currentNode := inputNode
	currentSize := inputSize
	weights := make([]*gorgonia.Node, len(hiddenLayers)+1)
	biases := make([]*gorgonia.Node, len(hiddenLayers)+1)

	// Hidden layers
	for i, layerSize := range hiddenLayers {
		// Weight matrix
		weights[i] = gorgonia.NewMatrix(g, tensor.Float64,
			gorgonia.WithShape(currentSize, layerSize),
			gorgonia.WithInit(gorgonia.GlorotU(1.0)),
			gorgonia.WithName(fmt.Sprintf("w%d", i)))

		// Bias vector
		biases[i] = gorgonia.NewVector(g, tensor.Float64,
			gorgonia.WithShape(layerSize),
			gorgonia.WithInit(gorgonia.Zeroes()),
			gorgonia.WithName(fmt.Sprintf("b%d", i)))

		// Linear transformation
		linear := gorgonia.Must(gorgonia.Mul(currentNode, weights[i]))
		withBias := gorgonia.Must(gorgonia.BroadcastAdd(linear, biases[i], nil, []byte{1}))

		// Activation function (ReLU for hidden layers)
		currentNode = gorgonia.Must(gorgonia.Rectify(withBias))
		currentSize = layerSize
	}

	// Output layer
	outputIdx := len(hiddenLayers)
	weights[outputIdx] = gorgonia.NewMatrix(g, tensor.Float64,
		gorgonia.WithShape(currentSize, outputSize),
		gorgonia.WithInit(gorgonia.GlorotU(1.0)),
		gorgonia.WithName("w_out"))

	biases[outputIdx] = gorgonia.NewVector(g, tensor.Float64,
		gorgonia.WithShape(outputSize),
		gorgonia.WithInit(gorgonia.Zeroes()),
		gorgonia.WithName("b_out"))

	linear := gorgonia.Must(gorgonia.Mul(currentNode, weights[outputIdx]))
	outputNode := gorgonia.Must(gorgonia.BroadcastAdd(linear, biases[outputIdx], nil, []byte{1}))

	// Create virtual machine
	machine := gorgonia.NewTapeMachine(g)

	nn := &NeuralNetwork{
		graph:        g,
		inputNode:    inputNode,
		outputNode:   outputNode,
		weights:      weights,
		biases:       biases,
		machine:      machine,
		learnRate:    config.LearningRate,
		hiddenLayers: hiddenLayers,
		inputSize:    inputSize,
		outputSize:   outputSize,
	}

	return nn, nil
}

// initializeRandomForest creates and initializes a random forest model
func (p *Predictor) initializeRandomForest(config ModelConfig) (*RandomForestModel, error) {
	// Create random forest classifier
	rf := ensemble.NewRandomForest(config.TreeCount, config.MaxDepth)
	
	model := &RandomForestModel{
		classifier:   rf,
		isRegression: true, // For regression tasks (ROI prediction)
	}

	return model, nil
}

// trainNeuralNetwork trains the neural network using gradient descent
func (p *Predictor) trainNeuralNetwork(trainData, valData *TrainingData, config ModelConfig) error {
	if p.neuralNetwork == nil {
		return fmt.Errorf("neural network not initialized")
	}

	g := p.neuralNetwork.graph
	batchSize := config.BatchSize
	if batchSize <= 0 {
		batchSize = 32
	}

	// Create labels node
	labelsNode := gorgonia.NewVector(g, tensor.Float64,
		gorgonia.WithShape(-1),
		gorgonia.WithName("labels"))

	// Define loss function (MSE for regression)
	diff := gorgonia.Must(gorgonia.Sub(p.neuralNetwork.outputNode, labelsNode))
	squared := gorgonia.Must(gorgonia.Square(diff))
	loss := gorgonia.Must(gorgonia.Mean(squared))

	// Create optimizer
	learningRate := p.neuralNetwork.learnRate
	if learningRate <= 0 {
		learningRate = 0.001
	}

	// Get trainable parameters
	var trainable gorgonia.Nodes
	trainable = append(trainable, p.neuralNetwork.weights...)
	trainable = append(trainable, p.neuralNetwork.biases...)

	solver := gorgonia.NewAdamSolver(gorgonia.WithLearnRate(learningRate))

	// Training loop
	for epoch := 0; epoch < config.Epochs; epoch++ {
		epochLoss := 0.0
		batches := len(trainData.Features) / batchSize

		for batch := 0; batch < batches; batch++ {
			start := batch * batchSize
			end := start + batchSize
			if end > len(trainData.Features) {
				end = len(trainData.Features)
			}

			// Prepare batch data
			batchFeatures := trainData.Features[start:end]
			batchLabels := trainData.Labels[start:end]

			// Convert to tensors
			featureTensor := p.createFeatureTensor(batchFeatures)
			labelTensor := tensor.New(tensor.WithBacking(batchLabels))

			// Set input values
			gorgonia.Let(p.neuralNetwork.inputNode, featureTensor)
			gorgonia.Let(labelsNode, labelTensor)

			// Forward and backward pass
			if err := p.neuralNetwork.machine.RunAll(); err != nil {
				return fmt.Errorf("forward pass failed: %w", err)
			}

			// Get loss value
			lossValue := loss.Value().Data().(float64)
			epochLoss += lossValue

			// Backward pass
			grad, err := loss.Grad()
			if err != nil {
				return fmt.Errorf("gradient computation failed: %w", err)
			}
			_ = grad // Use gradient variable to avoid unused variable error

			// Update parameters
			trainableVG := make([]gorgonia.ValueGrad, len(trainable))
			for i, node := range trainable {
				trainableVG[i] = node
			}
			if err := solver.Step(trainableVG); err != nil {
				return fmt.Errorf("parameter update failed: %w", err)
			}

			// Reset for next iteration
			p.neuralNetwork.machine.Reset()
		}

		// Log epoch progress
		if epoch%10 == 0 {
			avgLoss := epochLoss / float64(batches)
			p.logger.WithFields(logrus.Fields{
				"epoch":    epoch,
				"avg_loss": avgLoss,
			}).Debug("Neural network training progress")
		}
	}

	// Note: ExprGraph in newer versions doesn't have Close method
	// Memory is managed by Go's garbage collector

	return nil
}

// trainRandomForest trains the random forest model
func (p *Predictor) trainRandomForest(trainData *TrainingData, config ModelConfig) error {
	if p.randomForest == nil {
		return fmt.Errorf("random forest not initialized")
	}

	// Convert training data to GoLearn format
	dataset, err := p.convertToGoLearnFormat(trainData)
	if err != nil {
		return fmt.Errorf("data conversion failed: %w", err)
	}

	if denseDataset, ok := dataset.(*base.DenseInstances); ok {
		p.randomForest.dataset = denseDataset
	} else {
		return fmt.Errorf("dataset is not a DenseInstances type")
	}

	// Train the model
	p.randomForest.classifier.Fit(dataset)

	// Evaluate on training data
	predictions, err := p.randomForest.classifier.Predict(dataset)
	if err != nil {
		return fmt.Errorf("prediction failed during training: %w", err)
	}

	// Calculate training accuracy
	confusionMat, err := evaluation.GetConfusionMatrix(dataset, predictions)
	if err == nil {
		accuracy := evaluation.GetAccuracy(confusionMat)
		p.logger.WithFields(logrus.Fields{
			"training_accuracy": accuracy,
			"tree_count":       config.TreeCount,
		}).Info("Random forest training completed")
	}

	return nil
}

// predictNeuralNetwork generates predictions using the neural network
func (p *Predictor) predictNeuralNetwork(features []float64) (float64, float64, error) {
	if p.neuralNetwork == nil {
		return 0, 0, fmt.Errorf("neural network not initialized")
	}

	// Create feature tensor
	featureTensor := tensor.New(tensor.WithBacking(features), tensor.WithShape(1, len(features)))

	// Set input
	gorgonia.Let(p.neuralNetwork.inputNode, featureTensor)

	// Forward pass
	if err := p.neuralNetwork.machine.RunAll(); err != nil {
		return 0, 0, fmt.Errorf("prediction forward pass failed: %w", err)
	}

	// Get prediction
	output := p.neuralNetwork.outputNode.Value()
	prediction := output.Data().([]float64)[0]

	// Calculate confidence (simplified - could use uncertainty estimation)
	confidence := math.Min(0.95, math.Max(0.1, 1.0/(1.0+math.Abs(prediction))))

	// Reset machine
	p.neuralNetwork.machine.Reset()

	return prediction, confidence, nil
}

// predictRandomForest generates predictions using the random forest
func (p *Predictor) predictRandomForest(features []float64) (float64, float64, error) {
	if p.randomForest == nil {
		return 0, 0, fmt.Errorf("random forest not initialized")
	}

	// Create single-row dataset for prediction
	testData, err := p.createSingleRowDataset(features)
	if err != nil {
		return 0, 0, fmt.Errorf("test data creation failed: %w", err)
	}

	// Generate prediction
	predictions, err := p.randomForest.classifier.Predict(testData)
	if err != nil {
		return 0, 0, fmt.Errorf("prediction failed: %w", err)
	}

	// Extract prediction value
	prediction := p.extractPredictionValue(predictions, 0)

	// Calculate confidence based on tree agreement (simplified)
	confidence := 0.8 // Default confidence for random forest

	return prediction, confidence, nil
}

// Utility functions

func (p *Predictor) splitTrainingData(data *TrainingData, splitRatio float64) (*TrainingData, *TrainingData) {
	if splitRatio <= 0 || splitRatio >= 1 {
		return data, &TrainingData{} // Return all data as training if invalid split
	}

	splitIdx := int(float64(len(data.Features)) * (1.0 - splitRatio))

	trainData := &TrainingData{
		Features: data.Features[:splitIdx],
		Labels:   data.Labels[:splitIdx],
		UserIDs:  data.UserIDs[:splitIdx],
	}

	valData := &TrainingData{
		Features: data.Features[splitIdx:],
		Labels:   data.Labels[splitIdx:],
		UserIDs:  data.UserIDs[splitIdx:],
	}

	return trainData, valData
}

func (p *Predictor) featuresToArray(features map[string]float64) []float64 {
	// Convert feature map to fixed-size array
	// This should match the feature extraction order
	featureNames := []string{
		"total_lineups", "avg_score", "win_rate", "total_roi",
		"score_variance", "roi_variance", "avg_entry_fee",
		"stacking_usage_rate", "sport_diversity", "avg_ownership_selection",
		// Add more feature names as needed
	}

	result := make([]float64, len(featureNames))
	for i, name := range featureNames {
		if value, exists := features[name]; exists {
			result[i] = value
		}
	}

	return result
}

func (p *Predictor) createFeatureTensor(features [][]float64) *tensor.Dense {
	rows := len(features)
	cols := len(features[0])
	
	flat := make([]float64, rows*cols)
	for i, row := range features {
		copy(flat[i*cols:], row)
	}

	return tensor.New(tensor.WithBacking(flat), tensor.WithShape(rows, cols))
}

func (p *Predictor) convertToGoLearnFormat(data *TrainingData) (base.FixedDataGrid, error) {
	// Create temporary CSV file for GoLearn
	tmpFile, err := os.CreateTemp("", "training_data_*.csv")
	if err != nil {
		return nil, err
	}
	defer os.Remove(tmpFile.Name())

	writer := csv.NewWriter(tmpFile)
	defer writer.Flush()

	// Write header
	header := make([]string, len(data.Features[0])+1)
	for i := 0; i < len(data.Features[0]); i++ {
		header[i] = fmt.Sprintf("feature_%d", i)
	}
	header[len(header)-1] = "label"
	writer.Write(header)

	// Write data
	for i, features := range data.Features {
		row := make([]string, len(features)+1)
		for j, feature := range features {
			row[j] = strconv.FormatFloat(feature, 'f', -1, 64)
		}
		row[len(row)-1] = strconv.FormatFloat(data.Labels[i], 'f', -1, 64)
		writer.Write(row)
	}

	tmpFile.Close()

	// Load as GoLearn dataset
	dataset, err := base.ParseCSVToInstances(tmpFile.Name(), true)
	if err != nil {
		return nil, err
	}

	return dataset, nil
}

func (p *Predictor) createSingleRowDataset(features []float64) (base.FixedDataGrid, error) {
	// Create temporary CSV file
	tmpFile, err := os.CreateTemp("", "prediction_data_*.csv")
	if err != nil {
		return nil, err
	}
	defer os.Remove(tmpFile.Name())

	writer := csv.NewWriter(tmpFile)
	defer writer.Flush()

	// Write header
	header := make([]string, len(features)+1)
	for i := 0; i < len(features); i++ {
		header[i] = fmt.Sprintf("feature_%d", i)
	}
	header[len(header)-1] = "label"
	writer.Write(header)

	// Write single row
	row := make([]string, len(features)+1)
	for i, feature := range features {
		row[i] = strconv.FormatFloat(feature, 'f', -1, 64)
	}
	row[len(row)-1] = "0" // Dummy label
	writer.Write(row)

	tmpFile.Close()

	// Load as GoLearn dataset
	dataset, err := base.ParseCSVToInstances(tmpFile.Name(), true)
	if err != nil {
		return nil, err
	}

	return dataset, nil
}

func (p *Predictor) extractPredictionValue(predictions base.FixedDataGrid, row int) float64 {
	// Simple fallback - return a placeholder value
	// TODO: Implement proper GoLearn result extraction based on specific GoLearn version
	return 0.0
}