/**
 * DoYou Music Analytics - Scraper Integration Module
 * 
 * Add this to your scraper project to access analyzed track data.
 * Provides both REST API and direct database access methods.
 */

// ============================================
// METHOD 1: REST API (Recommended)
// ============================================

class AnalyticsClient {
  constructor(baseURL = 'http://localhost:8080') {
    this.baseURL = baseURL;
  }

  /**
   * Get statistics about analysis progress
   */
  async getStats() {
    const response = await fetch(`${this.baseURL}/api/v1/stats`);
    if (!response.ok) throw new Error(`API error: ${response.status}`);
    return response.json();
    // Returns: { total, pending, analyzing, completed, failed, missing }
  }

  /**
   * Get detailed statistics with performance metrics
   */
  async getDetailedStats() {
    const response = await fetch(`${this.baseURL}/api/v1/stats/detailed`);
    if (!response.ok) throw new Error(`API error: ${response.status}`);
    return response.json();
    // Returns: { ...stats, avg_processing_time_sec, tracks_per_hour, completion_percentage }
  }

  /**
   * Get tracks with optional filtering
   * @param {Object} options - Query options
   * @param {string} options.status - Filter by status (completed, pending, failed, missing)
   * @param {number} options.limit - Max results (default: 50)
   * @param {number} options.offset - Pagination offset (default: 0)
   */
  async getTracks(options = {}) {
    const params = new URLSearchParams();
    if (options.status) params.append('status', options.status);
    if (options.limit) params.append('limit', options.limit);
    if (options.offset) params.append('offset', options.offset);

    const response = await fetch(`${this.baseURL}/api/v1/tracks?${params}`);
    if (!response.ok) throw new Error(`API error: ${response.status}`);
    return response.json();
    /* Returns array of tracks:
    [{
      id: 123,
      file_name: "track.mp3",
      title: "Song Title",
      artist: "Artist Name",
      album: "Album",
      status: "completed",
      analyzed_at: "2026-02-13T10:30:00Z",
      audio_features: {
        bpm: 128.5,
        key: "A",
        mode: "Minor",
        duration: 245.3,
        energy: 0.85,
        danceability: 0.72,
        valence: 0.45,
        acousticness: 0.12,
        instrumentalness: 0.02,
        speechiness: 0.05,
        liveness: 0.18,
        melodicness: 0.68,
        loudness: -6.5
      }
    }]
    */
  }

  /**
   * Get single track by ID
   */
  async getTrack(id) {
    const response = await fetch(`${this.baseURL}/api/v1/tracks/${id}`);
    if (!response.ok) throw new Error(`API error: ${response.status}`);
    return response.json();
  }

  /**
   * Search tracks by title, artist, or album
   */
  async searchTracks(query) {
    const params = new URLSearchParams({ q: query });
    const response = await fetch(`${this.baseURL}/api/v1/tracks/search?${params}`);
    if (!response.ok) throw new Error(`API error: ${response.status}`);
    return response.json();
  }

  /**
   * Get all completed tracks with audio features
   */
  async getAnalyzedTracks(limit = 100) {
    return this.getTracks({ status: 'completed', limit });
  }

  /**
   * Check if specific file has been analyzed
   */
  async isFileAnalyzed(filename) {
    const results = await this.searchTracks(filename);
    const track = results.find(t => t.file_name === filename);
    return track && track.status === 'completed';
  }
}

// ============================================
// METHOD 2: Direct Database Access
// ============================================

const { Pool } = require('pg');

class AnalyticsDatabase {
  constructor(config = {}) {
    this.pool = new Pool({
      host: config.host || 'localhost',
      port: config.port || 5432,
      database: config.database || 'doyou_music',
      user: config.user || 'doyou',
      password: config.password || process.env.DB_PASSWORD
    });
  }

  /**
   * Get all completed tracks with audio features
   */
  async getAnalyzedTracks(limit = 100, offset = 0) {
    const result = await this.pool.query(`
      SELECT 
        t.id,
        t.file_name,
        t.file_path,
        t.title,
        t.artist,
        t.album,
        t.analyzed_at,
        af.bpm,
        af.key,
        af.mode,
        af.duration,
        af.energy,
        af.danceability,
        af.valence,
        af.acousticness,
        af.instrumentalness,
        af.speechiness,
        af.liveness,
        af.melodicness,
        af.loudness
      FROM tracks t
      JOIN audio_features af ON t.id = af.track_id
      WHERE t.analysis_status = 'completed'
      ORDER BY t.analyzed_at DESC
      LIMIT $1 OFFSET $2
    `, [limit, offset]);
    
    return result.rows;
  }

  /**
   * Check if file has been analyzed
   */
  async isFileAnalyzed(filename) {
    const result = await this.pool.query(
      `SELECT analysis_status FROM tracks WHERE file_name = $1`,
      [filename]
    );
    return result.rows.length > 0 && result.rows[0].analysis_status === 'completed';
  }

  /**
   * Get tracks by BPM range (for DJ mixing)
   */
  async getTracksByBPM(minBpm, maxBpm) {
    const result = await this.pool.query(`
      SELECT 
        t.file_name,
        t.artist,
        t.title,
        af.bpm,
        af.key,
        af.energy
      FROM tracks t
      JOIN audio_features af ON t.id = af.track_id
      WHERE t.analysis_status = 'completed'
        AND af.bpm BETWEEN $1 AND $2
      ORDER BY af.bpm
    `, [minBpm, maxBpm]);
    
    return result.rows;
  }

  /**
   * Get high-energy tracks for radio show
   */
  async getHighEnergyTracks(minEnergy = 0.7, limit = 50) {
    const result = await this.pool.query(`
      SELECT 
        t.file_name,
        t.artist,
        t.title,
        af.bpm,
        af.energy,
        af.danceability,
        af.valence
      FROM tracks t
      JOIN audio_features af ON t.id = af.track_id
      WHERE t.analysis_status = 'completed'
        AND af.energy > $1
      ORDER BY af.energy DESC
      LIMIT $2
    `, [minEnergy, limit]);
    
    return result.rows;
  }

  /**
   * Get analysis progress stats
   */
  async getStats() {
    const result = await this.pool.query(`
      SELECT 
        analysis_status,
        COUNT(*) as count
      FROM tracks
      GROUP BY analysis_status
    `);
    
    const stats = {};
    result.rows.forEach(row => {
      stats[row.analysis_status] = parseInt(row.count);
    });
    
    return stats;
  }

  async close() {
    await this.pool.end();
  }
}

// ============================================
// USAGE EXAMPLES
// ============================================

async function exampleUsage() {
  // Method 1: REST API
  console.log('=== Using REST API ===');
  const api = new AnalyticsClient();
  
  const stats = await api.getStats();
  console.log(`Progress: ${stats.completed}/${stats.total} tracks analyzed`);
  
  const analyzedTracks = await api.getAnalyzedTracks(10);
  console.log(`First 10 analyzed tracks:`, analyzedTracks.map(t => t.file_name));
  
  const isAnalyzed = await api.isFileAnalyzed('some-track.mp3');
  console.log(`Track analyzed: ${isAnalyzed}`);

  // Method 2: Direct Database
  console.log('\n=== Using Direct Database ===');
  const db = new AnalyticsDatabase();
  
  const tracks = await db.getAnalyzedTracks(5);
  console.log(`Found ${tracks.length} analyzed tracks`);
  
  const highEnergy = await db.getHighEnergyTracks(0.8, 10);
  console.log(`High energy tracks:`, highEnergy);
  
  const bpmRange = await db.getTracksByBPM(120, 130);
  console.log(`Tracks at 120-130 BPM:`, bpmRange);
  
  await db.close();
}

// ============================================
// EXPORT
// ============================================

module.exports = {
  AnalyticsClient,
  AnalyticsDatabase
};

// Example: Add to your scraper's package.json scripts:
// "check-analysis": "node -e \"require('./analytics-client').exampleUsage()\""
