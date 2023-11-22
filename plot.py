import pandas as pd
import numpy as np
import matplotlib.pyplot as plt
import seaborn as sns

# Read in the saved CSV data and save into columns
benchmark_data = pd.read_csv('results.csv', header=0, names=['name', 'time', 'range'])
benchmark_data['percentage'] = benchmark_data['range'].str.extract('(\d+)').astype(float)
benchmark_data['error'] = (benchmark_data['time'] / 100) * benchmark_data['percentage']
benchmark_data['threads'] = benchmark_data['name'].str.extract('Gol/\d+x\d+x\d+-(\d+)-\d+').apply(pd.to_numeric)

print(benchmark_data)

# Plot a bar chart.
# Create the main bar chart using ax.bar with explicit order
ax = plt.gca()

# Custom bar plot using ax.bar
bars = ax.bar(benchmark_data['threads'], benchmark_data['time'], color=sns.color_palette("husl", n_colors=len(benchmark_data['threads'])))

# Add error bars using ax.errorbar
for bar, (_, row) in zip(bars, benchmark_data.iterrows()):
    ax.errorbar(x=bar.get_x() + bar.get_width() / 2, y=row['time'], yerr=row['error'], color='black', capsize=6, capthick=1, fmt='none')
# Set descriptive axis lables.
ax.set(xlabel='Worker threads used', ylabel='Mean time taken (s)')


# Display the full figure.
plt.show()
