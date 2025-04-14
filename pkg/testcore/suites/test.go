// Occasionally the sha512sum function returns empty even when the driver has correctly written data. This retry logic allows the sha512sum function to run again and correctly calculate the hash value.
maxRetries := 3
retryCount := 0
for retryCount < maxRetries {
	writer := bytes.NewBufferString("")
	if err := podClient.Exec(ctx, writerPod.Object, []string{"/bin/bash", "-c", "sha512sum -c " + sum}, writer, os.Stderr, false); err != nil {
		return err
	}

	if strings.Contains(writer.String(), "OK") {
		log.Infof("Hashes match. Writer content: %s, Sum file: %s", writer.String(), sum)
		break
	}
	log.Infof("Hashes don't match. Retrying... Writer content: %s, Sum file: %s", writer.String(), sum)
	retryCount++
	time.Sleep(2 * time.Second)
}
if retryCount == maxRetries {
	return fmt.Errorf("Max number of retries reached, hashes don't match")
}

// Occasionally the sha512sum function returns empty even when the driver has correctly written data. This retry logic allows the sha512sum function to run again and correctly calculate the hash value.
maxRetries := 3
retryCount := 0
for retryCount < maxRetries {
	log.Infof("Iteration number: %d", retryCount)

	if err := podClient.Exec(ctx, writerPod.Object, []string{"/bin/bash", "-c", "sha512sum -c " + sum}, writer, os.Stderr, false); err != nil {
		return delFunc, err
	}

	if strings.Contains(writer.String(), "OK") {
		log.Infof("Hashes match. Writer content: %s, Sum file: %s", writer.String(), sum)
		break
	}
	log.Infof("Hashes don't match. Retrying... Writer content: %s, Sum file: %s", writer.String(), sum)
	retryCount++
	time.Sleep(2 * time.Second)
}
if retryCount == maxRetries {
	log.Error("Max retries reached, hash sum is still empty")
}

// Occasionally the sha512sum function returns empty even when the driver has correctly written data. This retry logic allows the sha512sum function to run again and correctly calculate the hash value.
maxRetries := 3
retryCount := 0
for retryCount < maxRetries {
	// Calculate hash sum of that file
	if err := podClient.Exec(ctx, originalPod.Object, []string{"sha512sum", file}, hash, os.Stderr, false); err != nil {
		return delFunc, err
	}

	if hash.String() != "" {
		log.Info("OriginalPod: ", originalPod.Object.GetName())
		log.Info("hash sum is:", hash.String())
		break
	}

	log.Info("hash sum is empty, retrying...")

	retryCount++
	time.Sleep(2 * time.Second)
}

if retryCount == maxRetries {
	return delFunc, fmt.Errorf("max number of retries reached, original hash is empty")
}


// Occasionally the sha512sum function returns empty even when the driver has correctly written data. This retry logic allows the sha512sum function to run again and correctly calculate the hash value.
maxRetries := 3
retryCount := 0
for retryCount < maxRetries {
	if err := podClient.Exec(ctx, p.Object, []string{"sha512sum", file}, newHash, os.Stderr, false); err != nil {
		return delFunc, err
	}

	if newHash.String() != "" {
		break
	}

	log.Info("Pod: ", p.Object.GetName())
	log.Info("hash sum is empty, retrying...")

	retryCount++
	time.Sleep(2 * time.Second)
}

if retryCount == maxRetries {
	return delFunc, fmt.Errorf("max number of retries reached, hashes don't match")
}